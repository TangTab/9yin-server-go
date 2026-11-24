package gmpanel

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/scenedata"
)

//go:embed web
var assets embed.FS

// routes API + 静态页装配。
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/overview", s.apiOverview)
	mux.HandleFunc("/api/logs", s.apiLogs)
	mux.HandleFunc("/api/players", s.apiPlayers)
	mux.HandleFunc("/api/roles", s.apiRoles)
	mux.HandleFunc("/api/role", s.apiRoleDetail)
	mux.HandleFunc("/api/alerts", s.apiAlerts)
	mux.HandleFunc("/api/svc/", s.apiSvc)
	mux.HandleFunc("/api/gm/scenes", s.apiGMScenes)
	mux.HandleFunc("/api/gm/", s.apiGM)
	mux.HandleFunc("/api/world/", s.apiWorld)
	mountWeb(mux)
	return mux
}

/* ---------------- 通用 ---------------- */

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

// postBody 解析 {name}/{account} 单键 JSON 体。
func postBody(r *http.Request) map[string]string {
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body == nil {
		body = map[string]string{}
	}
	return body
}

// noCacheStatic 静态资源 no-cache（embed.FS 无 Last-Modified/ETag，
// 浏览器启发式缓存会把模块缓存死——wt1653 线实证坑，照搬 gmweb.go）。
func noCacheStatic(files http.FileSystem) http.Handler {
	fileServer := http.StripPrefix("/static/", http.FileServer(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}

// mountWeb "/" 直出 index.html，"/static/" 映射嵌入的 web/static。
func mountWeb(mux *http.ServeMux) {
	static, err := fs.Sub(assets, "web/static")
	if err != nil {
		return
	}
	mux.Handle("/static/", noCacheStatic(http.FS(static)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		content, err := assets.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "panel index missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(content)
	})
}

/* ---------------- 概览 ---------------- */

func (s *Server) apiOverview(w http.ResponseWriter, r *http.Request) {
	world := map[string]any{"alive": false}
	if lines, err := s.worldCli().Command("status"); err == nil && len(lines) > 0 {
		world["alive"], world["state"] = true, lines[0]
	}
	online := -1
	if lines, err := s.worldCli().Command("players"); err == nil && len(lines) > 0 {
		online = parsePlayerCounts(lines[0], world)
	}
	dbOK := false
	if s.cfg.DSN != "" {
		if _, err := s.store(); err == nil {
			dbOK = true
		}
	}
	mem := s.sys.Info()
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":     s.cfg.Mode,
		"services": s.sup.StatusAll(),
		"world":    world,
		"online":   online,
		"db":       dbOK,
		"sys": map[string]any{
			"memLoad": mem.MemLoad, "totalPhy": mem.TotalPhy, "availPhy": mem.AvailPhy,
		},
		"time": time.Now().Format("15:04:05"),
	})
}

// parsePlayerCounts 解析 `player: N, online: N, offline: N, active: N,
// queuing: N`（echo players 应答），online 数回填 world["counts"]。
func parsePlayerCounts(line string, world map[string]any) int {
	world["counts"] = line
	for _, seg := range strings.Split(line, ",") {
		kv := strings.SplitN(strings.TrimSpace(seg), ":", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "online" {
			n, _ := strconv.Atoi(strings.TrimSpace(kv[1]))
			return n
		}
	}
	return -1
}

/* ---------------- 日志 ---------------- */

func (s *Server) apiLogs(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	tail := 400
	if v, err := strconv.Atoi(r.URL.Query().Get("tail")); err == nil && v > 0 && v <= 5000 {
		tail = v
	}
	if s.sup.Spec(name) == nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("unknown service %q", name))
		return
	}
	path := s.sup.LatestLog(name)
	if path == "" {
		writeJSON(w, http.StatusOK, map[string]any{"name": name, "file": "", "lines": []string{}})
		return
	}
	lines, err := tailLines(path, tail)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "file": path, "lines": lines})
}

// tailLines 读文件尾部 n 行（本地日志体量小，整读切分即可；进程写侧
// 持有句柄不阻塞共享读，半行容忍）。
func tailLines(path string, n int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = []byte(strings.TrimSuffix(string(data), "\n"))
	if len(data) == 0 {
		return []string{}, nil
	}
	all := strings.Split(string(data), "\n")
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all, nil
}

/* ---------------- 玩家（world echo plist） ---------------- */

type playerRow struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Scene   int    `json:"scene"`
	LoginAt int64  `json:"loginAt"`
	ActiveAt int64 `json:"activeAt"`
}

func (s *Server) apiPlayers(w http.ResponseWriter, r *http.Request) {
	lines, err := s.worldCli().Command("plist")
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	rows := make([]playerRow, 0, len(lines))
	for _, ln := range lines {
		if strings.HasPrefix(ln, "player ") {
			continue // 首行 `player N` 计数头
		}
		f := strings.Split(ln, "|")
		if len(f) < 6 {
			continue
		}
		row := playerRow{Account: f[0], Name: f[1], State: f[2]}
		row.Scene, _ = strconv.Atoi(f[3])
		row.LoginAt, _ = strconv.ParseInt(f[4], 10, 64)
		row.ActiveAt, _ = strconv.ParseInt(f[5], 10, 64)
		rows = append(rows, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"players": rows})
}

/* ---------------- 角色 / 告警（MySQL） ---------------- */

func (s *Server) apiRoles(w http.ResponseWriter, r *http.Request) {
	account := strings.TrimSpace(r.URL.Query().Get("account"))
	if account == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("account 参数必填"))
		return
	}
	db, err := s.store()
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	rows, err := db.SearchRoles(account)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": account, "roles": rows})
}

func (s *Server) apiRoleDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	account, name := strings.TrimSpace(q.Get("account")), strings.TrimSpace(q.Get("name"))
	serverID, _ := strconv.Atoi(q.Get("server"))
	if account == "" || name == "" || serverID == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("account/name/server 参数必填"))
		return
	}
	db, err := s.store()
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	_, meta, ok, err := db.LoadRole(account, name, serverID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, fmt.Errorf("角色不存在"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"meta": meta})
}

func (s *Server) apiAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 1000 {
		limit = v
	}
	db, err := s.store()
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	rows, err := db.RecentAlerts(limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"alerts": rows})
}

/* ---------------- 服务启停 ---------------- */

func (s *Server) apiSvc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, fmt.Errorf("POST only"))
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/api/svc/")
	var msgs []string
	var err error
	switch action {
	case "start", "stop", "restart":
		name := postBody(r)["name"]
		var msg string
		switch action {
		case "start":
			msg, err = s.sup.Start(name)
		case "stop":
			msg, err = s.sup.Stop(name)
		default:
			msg, err = s.sup.Restart(name)
		}
		msgs = []string{msg}
	case "startall":
		msgs = s.sup.StartAll()
	case "stopall":
		msgs = s.sup.StopAll()
	default:
		writeErr(w, http.StatusNotFound, fmt.Errorf("unknown action %q", action))
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}

/* ---------------- GM 动作 ---------------- */

func (s *Server) apiGM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, fmt.Errorf("POST only"))
		return
	}
	action := strings.TrimPrefix(r.URL.Path, "/api/gm/")
	body := postBody(r)
	switch action {
	case "raw":
		// GM 测试台透传：仅放行 world echo 的 `gm ` 前缀命令族
		//（gmcmd.go：state/set/money/tp/msg/view/flush/save）。
		line := strings.TrimSpace(body["line"])
		if line == "" || !strings.HasPrefix(line, "gm ") {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("仅允许 gm 前缀命令（gm state/set/money/tp/msg/view/flush/save …）"))
			return
		}
		lines, err := s.worldCli().Command(line)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
	case "kick":
		name := strings.TrimSpace(body["name"])
		if name == "" {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("name 必填"))
			return
		}
		lines, err := s.worldCli().Command("kick " + name)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err)
			return
		}
		msg := strings.Join(lines, "；")
		if strings.Contains(msg, "not found") {
			writeErr(w, http.StatusNotFound, fmt.Errorf("玩家 %s 不在线或不存在", name))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": msg})
	case "ban", "unban":
		account := strings.TrimSpace(body["account"])
		if account == "" {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("account 必填"))
			return
		}
		status := 0 // AccountLogon 登录门 status=1；封号置 0
		if action == "unban" {
			status = 1
		}
		db, err := s.store()
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err)
			return
		}
		ok, err := db.SetAccountStatus(account, status)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, fmt.Errorf("账号 %s 不存在", account))
			return
		}
		// 封号连带踢下线（在线账号不残留会话）；解封无需踢
		kickMsg := ""
		if status == 0 {
			if lines, err := s.worldCli().Command("kick " + account); err == nil {
				kickMsg = strings.Join(lines, "；")
			}
		}
		msg := "账号 " + account + " 已" + map[bool]string{true: "解封", false: "封禁"}[status == 1]
		if kickMsg != "" && !strings.Contains(kickMsg, "not found") {
			msg += "；" + kickMsg
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": msg})
	default:
		writeErr(w, http.StatusNotFound, fmt.Errorf("unknown action %q", action))
	}
}

/* ---------------- world 开关服 ---------------- */

func (s *Server) apiWorld(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, fmt.Errorf("POST only"))
		return
	}
	cmd := strings.TrimPrefix(r.URL.Path, "/api/world/")
	if cmd != "open" && cmd != "close" {
		writeErr(w, http.StatusNotFound, fmt.Errorf("unknown action %q", cmd))
		return
	}
	lines, err := s.worldCli().Command(cmd)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": strings.Join(lines, "；")})
}

/* ---------------- GM 场景表 ---------------- */

// apiGMScenes GM 测试台「传送与场景」下拉数据：scenedata 场景/驿站表（与
// world 同一份内嵌生成表——下拉列的每个场景 world 都能解析，不再手维
// JS 预设表）。按类别分组、显示中文名；驿站供按点传送（gm scene <who>
// <key> x y z orient 的坐标覆盖）。
func (s *Server) apiGMScenes(w http.ResponseWriter, r *http.Request) {
	table := scenedata.Scenes()
	scenes := make([]map[string]any, 0, len(table))
	for _, sc := range table {
		waypoints := make([]map[string]any, 0, len(sc.Waypoints))
		for _, wp := range sc.Waypoints {
			waypoints = append(waypoints, map[string]any{
				"name": wp.Name, "title": wp.Title, "x": wp.X, "y": wp.Y, "z": wp.Z,
				"orient": wp.Orient, "kind": wp.Kind, "safe": wp.Safe,
			})
		}
		scenes = append(scenes, map[string]any{
			"key": sc.Resource, "name": sc.Name, "category": sc.Category,
			"categoryName": scenedata.CategoryName(sc.Category), "config": sc.Config,
			"landing": sc.Landing.Name, "waypoints": waypoints,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes})
}
