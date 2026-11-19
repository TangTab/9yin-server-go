// genlic 是 license 生成与校验工具（FsEntry/FsSoloEntry 家族的
// `<外网IP>.lic` 方言，双方言二选一）。
//
// 用法：
//
//	genlic                       # solo 方言：自动探测本机 IP → 内部形态 <ip>.lic
//	genlic -ip 127.0.0.1         # 指定 IP 生成内部形态（256B）
//	genlic -ip 1.2.3.4 -external -days 30
//	                              # solo 外部形态（272B，时间窗=现在起 30 天）
//	genlic -ip 1.2.3.4 -external -t0 2026-11-19 -t1 2027-11-19
//	                              # 显式时间窗（日期或 unix 秒）
//	genlic -gw -ip 192.168.1.10  # 网关方言（FsEntry.dll：256B 单形态，
//	                              # 魔数对 0x8E74C52F/0xEB64EB5C）
//	genlic -verify 127.0.0.1.lic # 校验并解析已有 lic（-gw 切方言）
//
// 两种方言（内部包全解）：
//   - solo（缺省，internal/fssoloentry）：3DES-ECB（md5(盐16||IP截16B)
//     派生），身份域 8×u32（Fields[0]/[1]=魔数对 0xE7990369/0x87930363），
//     0xFC 处前 252 字节求和校验；内部形态 256B 仅限 127.0.0.1/192.168.*，
//     外部形态 272B 尾部 16B 二段密钥载体 + [t0,t1] 时间窗；
//   - 网关（-gw，internal/entry）：md5(IP+盐16+0xCC×16) 派生，256B 单
//     形态，校验只认首块魔数对（官方快照 127.0.0.1.lic 即此方言）。
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloentry"
)

const version = "0.2.0-genlic"

func main() {
	var (
		gw       = flag.Bool("gw", false, "网关方言（FsEntry.dll：256B 单形态，魔数对 0x8E74C52F/0xEB64EB5C）")
		ip       = flag.String("ip", "", "目标 IP（空=自动探测本机 IP；solo 内部形态须 127.0.0.1 或 192.168.*）")
		out      = flag.String("out", "", "输出路径（空=<ip>.lic）")
		external = flag.Bool("external", false, "solo 外部形态（272B，带时间窗；-gw 下无效）")
		days     = flag.Int("days", 30, "外部形态有效天数（从现在起算）")
		t0s      = flag.String("t0", "", "时间窗起点（unix 秒或 2006-01-02[ 15:04:05]；覆盖 -days）")
		t1s      = flag.String("t1", "", "时间窗终点（同上；缺省=t0+days）")
		verify   = flag.String("verify", "", "只校验并解析已有 lic 文件（值=路径）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go genlic %s（license 生成/校验工具：%s）\n", version,
		map[bool]string{true: "网关方言", false: "solo 方言"}[*gw])

	target := fssoloentry.LocalIPString(*ip)
	if *ip == "" {
		fmt.Printf("自动探测本机 IP：%s\n", target)
	}
	path := pickPath(*out, target)

	if *verify != "" {
		os.Exit(verifyOnly(*gw, *verify, target))
	}

	// 生成
	var data []byte
	if *gw {
		data = entry.GenerateLicense(target, entry.OfficialMagic1, entry.OfficialMagic2)
		if err := os.WriteFile(path, data, 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("已生成 %s（网关方言 256B，魔数对 %#X/%#X）\n", path,
			entry.OfficialMagic1, entry.OfficialMagic2)
	} else if *external {
		t0, t1, err := parseWindow(*t0s, *t1s, *days)
		if err != nil {
			fatal(err)
		}
		data = fssoloentry.GenerateExternalLicense(target, uint32(t0), uint32(t1))
		if err := os.WriteFile(path, data, 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("已生成 %s（外部形态 272B，魔数对 %#X/%#X）\n", path,
			fssoloentry.OfficialMagic1, fssoloentry.OfficialMagic2)
		fmt.Printf("时间窗：%s → %s\n",
			time.Unix(t0, 0).Format("2006-01-02 15:04:05"),
			time.Unix(t1, 0).Format("2006-01-02 15:04:05"))
	} else {
		data = fssoloentry.GenerateLicense(target)
		if err := os.WriteFile(path, data, 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("已生成 %s（内部形态 256B，魔数对 %#X/%#X）\n", path,
			fssoloentry.OfficialMagic1, fssoloentry.OfficialMagic2)
	}

	// 生成后回环自检（decode 走官方校验序：尺寸门/IP 门槛/求和/时间窗）
	if err := roundtrip(*gw, path, target); err != nil {
		fmt.Fprintf(os.Stderr, "genlic: 回环自检失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Println("回环自检通过（官方校验序全过）")
}

// verifyOnly 校验并打印 lic；返回退出码（0=Licensed；1=解码失败；2=魔数失配）。
func verifyOnly(gw bool, path, ip string) int {
	if gw {
		id, err := entry.LoadLicenseFile(path, ip, 0, 0) // 仅解出不校验
		if err != nil {
			fmt.Fprintf(os.Stderr, "genlic: %v\n", err)
			return 1
		}
		printGateway(path, ip, id)
		if id.Magic1 != entry.OfficialMagic1 || id.Magic2 != entry.OfficialMagic2 {
			fmt.Println("魔数对失配：按未授权处理")
			return 2
		}
		fmt.Println("判定：Licensed（首块魔数对命中）")
		return 0
	}
	lic, err := fssoloentry.LoadLicenseFile(path, ip, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "genlic: %v\n", err)
		return 1
	}
	printSolo(path, ip, lic)
	if !lic.Licensed() {
		fmt.Println("魔数对失配：按未授权处理（entry 不广播 + 每拍告警）")
		return 2
	}
	return 0
}

// roundtrip 生成后回环：按各方言官方校验序重新解码。
func roundtrip(gw bool, path, ip string) error {
	if gw {
		_, err := entry.LoadLicenseFile(path, ip, entry.OfficialMagic1, entry.OfficialMagic2)
		return err
	}
	lic, err := fssoloentry.LoadLicenseFile(path, ip, time.Now())
	if err != nil {
		return err
	}
	if !lic.Licensed() {
		return fmt.Errorf("魔数对失配")
	}
	printSolo(path, ip, lic)
	return nil
}

// printSolo 打印 solo 方言解码结果（身份域 + 形态 + 时间窗）。
func printSolo(path, ip string, lic fssoloentry.License) {
	form := "外部形态"
	if lic.Internal {
		form = "内部形态"
	}
	fmt.Printf("%s：IP %s，%s，身份域：\n", path, ip, form)
	for i, v := range lic.Fields {
		mark := ""
		switch i {
		case 0:
			mark = fmt.Sprintf("（魔数1 %#X %s）", v,
				map[bool]string{true: "命中", false: "失配"}[v == fssoloentry.OfficialMagic1])
		case 1:
			mark = fmt.Sprintf("（魔数2 %#X %s）", v,
				map[bool]string{true: "命中", false: "失配"}[v == fssoloentry.OfficialMagic2])
		default:
			mark = fieldMark(v)
		}
		fmt.Printf("  Fields[%d] = %#08X %s\n", i, v, mark)
	}
	if !lic.Internal {
		now := uint32(time.Now().Unix())
		state := "窗口内"
		switch {
		case now < lic.T0:
			state = "too young（未生效）"
		case now > lic.T1:
			state = "too old（已过期）"
		}
		fmt.Printf("时间窗：%s → %s（%s，剩余 %d 天）\n",
			time.Unix(int64(lic.T0), 0).Format("2006-01-02 15:04:05"),
			time.Unix(int64(lic.T1), 0).Format("2006-01-02 15:04:05"),
			state, (int64(lic.T1)-time.Now().Unix())/86400)
	}
	if lic.Licensed() {
		fmt.Println("判定：Licensed（TimerCheck 门槛通过）")
	}
}

// printGateway 打印网关方言解码结果（首块 20B 身份域）。
func printGateway(path, ip string, id entry.LicenseIdentity) {
	fmt.Printf("%s：IP %s，网关方言 256B，首块身份域：\n", path, ip)
	fmt.Printf("  Magic1 = %#08X（%s）\n", id.Magic1,
		map[bool]string{true: "命中", false: "失配"}[id.Magic1 == entry.OfficialMagic1])
	fmt.Printf("  Magic2 = %#08X（%s）\n", id.Magic2,
		map[bool]string{true: "命中", false: "失配"}[id.Magic2 == entry.OfficialMagic2])
	for i, v := range id.Spare {
		fmt.Printf("  Spare[%d] = %#08X %s\n", i, v, fieldMark(v))
	}
}

// fieldMark 身份域填充字的呈现注记（官方语义未定案，按位呈现）。
func fieldMark(v uint32) string {
	if v == 0 {
		return "（零填充）"
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return fmt.Sprintf("（% X）", b[:])
}

// parseWindow 解析时间窗：-t0/-t1 显式（unix 秒或日期）或 -days 推算。
func parseWindow(t0s, t1s string, days int) (int64, int64, error) {
	now := time.Now()
	if t0s == "" && t1s == "" {
		if days <= 0 {
			return 0, 0, fmt.Errorf("-days 须为正数（得 %d）", days)
		}
		return now.Unix(), now.AddDate(0, 0, days).Unix(), nil
	}
	t0, err := parseTime(t0s, now.Unix())
	if err != nil {
		return 0, 0, fmt.Errorf("-t0: %w", err)
	}
	var t1 int64
	if t1s == "" {
		t1 = time.Unix(t0, 0).AddDate(0, 0, days).Unix()
	} else if t1, err = parseTime(t1s, t0); err != nil {
		return 0, 0, fmt.Errorf("-t1: %w", err)
	}
	if t1 <= t0 {
		return 0, 0, fmt.Errorf("时间窗倒置：t0=%d t1=%d", t0, t1)
	}
	return t0, t1, nil
}

// parseTime 接受 unix 秒或 2006-01-02[ 15:04:05]（本地时区）。
func parseTime(s string, def int64) (int64, error) {
	if s == "" {
		return def, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Unix(), nil
		}
	}
	return 0, fmt.Errorf("无法解析时间 %q（unix 秒或 2006-01-02[ 15:04:05]）", s)
}

func pickPath(out, ip string) string {
	if out != "" {
		return out
	}
	return ip + ".lic"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "genlic:", err)
	os.Exit(1)
}
