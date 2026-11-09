# -*- coding: utf-8 -*-
r"""按部署资源刷新 game_items 迁移（0002 口径的增量再生成）。

数据源:F:\Game\Online\9yin\9yin-server-go\resources\modern\share\item 的
tool_item.ini + equipment.ini（GBK）。生成口径与 migrations/0002 逐字段
零错配验证过(2026-11-09,105,034 行):

  归类   equipment.ini → 'equip';tool_item.ini 按 ViewID 1→tool/2→equip/
        3→material/4→task(其余→tool);
  取值   整数列取"值的前导整数"(畸形行 MaxHardiness=100␣␣Name=藏风 取 100);
  排除   '//' 开头的注释节;
  伪影   仅 item_name 差异且 ini 字节未变的行(UTF-8 字节混在 GBK 文件里,
        整体按 GBK 解码的替换位差异)跳过不动。

用法:
  python tools/game_items_refresh.py            # 差异报告(dry-run)
  python tools/game_items_refresh.py --write NNNN_game_items_refresh_YYYYMMDD

产物为 tar+gzip 单成员归档(export-sql.bat 打包约定,runner 按内部 SQL 文本
计算校验和),幂等 REPLACE,经 cmd/sql-replay 回放。

--fix-names 模式:占位名(纯数字编号/空/==item_id)按
text/chineses/stringname.idres(小写键,UTF-8)修真名,口径同运行时
gmItemDisplayName 的 idres 优先链;全行取自活库仅改 item_name,其余 15 列
零扰动。DB 连接取 NINEYIN_MYSQL_DSN(默认 nineyin@127.0.0.1:3306/nineyin)。

用法:
  python tools/game_items_refresh.py            # 差异报告(dry-run)
  python tools/game_items_refresh.py --write NNNN_game_items_refresh_YYYYMMDD
  python tools/game_items_refresh.py --fix-names NNNN_game_items_names_YYYYMMDD
"""
import io
import os
import re
import sys
import tarfile

F_ITEM = r"F:\Game\Online\9yin\9yin-server-go\resources\modern\share\item"
F_IDRES = r"F:\Game\Online\9yin\9yin-server-go\resources\modern\text\chineses\stringname.idres"
DUMP_0002 = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
                         "migrations", "0002_config_data_20261031.sql.gz")
SRC_TOOL = "resources\\modern\\share\\item\\tool_item.ini"
SRC_EQUIP = "resources\\modern\\share\\item\\equipment.ini"

COLS = ["item_id", "item_category", "item_name", "item_type", "view_id",
        "equip_type", "script", "max_amount", "func_pack", "logic_pack",
        "prop_modify_pack", "texture_type", "art_pack", "hardiness",
        "max_hardiness", "source_file"]
STR_COLS = {0, 1, 2, 5, 6, 15}
VIEW_CAT = {1: "tool", 2: "equip", 3: "material", 4: "task"}


def parse_ini(path, src_file):
    """GBK ini → {节名: 16 列 list}，category 占位由 load_rows 统一回填。"""
    data = open(path, "rb").read().decode("gbk", errors="replace")
    rows, order, cur = {}, [], None
    for raw in data.splitlines():
        line = raw.strip()
        if not line or line.startswith((";", "#")):
            continue
        if line.startswith("[") and line.endswith("]"):
            cur = line[1:-1].strip()
            if cur and cur not in rows:
                rows[cur] = {}
                order.append(cur)
            continue
        if cur is None or "=" not in line:
            continue
        k, v = line.split("=", 1)
        rows[cur][k.strip().lower()] = v.strip()
    out = {}
    for sid in order:
        kv = rows[sid]

        def ival(key, _kv=kv):
            m = re.match(r"[-+]?\d+", (_kv.get(key.lower(), "") or "").strip())
            return int(m.group(0)) if m else 0
        out[sid] = [
            sid, "", kv.get("name", ""), ival("ItemType"), ival("ViewID"),
            kv.get("equiptype", ""), kv.get("script", ""),
            ival("MaxAmount"), ival("FuncPack"), ival("LogicPack"),
            ival("PropModifyPack"), ival("TextureType"), ival("ArtPack"),
            ival("Hardiness"), ival("MaxHardiness"), src_file,
        ]
    return out


def load_ini_rows(item_root=F_ITEM):
    rows = {}
    rows.update(parse_ini(os.path.join(item_root, "equipment.ini"), SRC_EQUIP))
    rows.update(parse_ini(os.path.join(item_root, "tool_item.ini"), SRC_TOOL))
    for sid, r in rows.items():
        r[1] = "equip" if r[15] == SRC_EQUIP else VIEW_CAT.get(r[4], "tool")
    return {k: v for k, v in rows.items() if not k.startswith("//")}


def parse_dump_rows(dump_path=DUMP_0002):
    """mysqldump 风格多语句 → {item_id: 16 列}，逐字符切元组尊重引号转义。"""
    import gzip
    with gzip.open(dump_path, "rt", encoding="utf-8", errors="replace") as f:
        text = f.read()
    stmts = re.findall(r"REPLACE INTO `game_items` VALUES (.*?);\n", text, re.S)
    if not stmts:
        raise SystemExit("dump 中未找到 game_items 段: %s" % dump_path)
    body = "\n".join(stmts)
    rows = {}
    i, n = 0, len(body)
    while i < n:
        if body[i] != "(":
            i += 1
            continue
        j, fields, buf, in_str = i + 1, [], [], False
        while j < n:
            ch = body[j]
            if in_str:
                if ch == "\\" and j + 1 < n:
                    buf.append({"n": "\n", "r": "\r", "t": "\t", "0": "\0"}.get(body[j + 1], body[j + 1]))
                    j += 2
                    continue
                if ch == "'":
                    in_str = False
                else:
                    buf.append(ch)
            elif ch == "'":
                in_str = True
            elif ch == ",":
                fields.append("".join(buf))
                buf = []
            elif ch == ")":
                fields.append("".join(buf))
                break
            elif ch.strip():
                buf.append(ch)
            j += 1
        if len(fields) == 16:
            typed = [int(f) if k not in STR_COLS else f for k, f in enumerate(fields)]
            rows[typed[0]] = typed
        i = j + 1
    return rows


def compute_refresh(baseline=None):
    """→ (新增行, 真实变更行, 伪影 item_id 列表, 移除 item_id 列表, 基线行数)。

    基线默认 0002 dump;差异集仅接受纯 func_pack 变更,其余字段差异报错,
    item_name-only 差异按伪影跳过(ini 字节未变,解码器替换位不同)。
    """
    baseline = baseline if baseline is not None else parse_dump_rows()
    ini = load_ini_rows()
    added = [ini[sid] for sid in sorted(set(ini) - set(baseline))]
    removed = sorted(set(baseline) - set(ini))
    changed, artifacts = [], []
    for sid in sorted(set(ini) & set(baseline)):
        a, b = baseline[sid], ini[sid]
        diff = {COLS[k] for k in range(16) if a[k] != b[k]}
        if not diff:
            continue
        if diff <= {"item_name"}:
            artifacts.append(sid)
        elif diff <= {"func_pack"}:
            changed.append(b)
        else:
            raise SystemExit("未预期差异 %s: %s" % (sid, {c: (a[k], b[k]) for k, c in enumerate(COLS) if c in diff}))
    return added, changed, artifacts, removed, len(baseline)


def sql_str(v):
    return "'" + v.replace("\\", "\\\\").replace("'", "\\'") \
                  .replace("\n", "\\n").replace("\r", "\\r").replace("\0", "\\0") + "'"


def tuple_sql(r):
    return "(" + ",".join(sql_str(v) if k in STR_COLS else str(v) for k, v in enumerate(r)) + ")"


def write_migration(rows, out_path, inner_name, chunk=400):
    buf = io.StringIO()
    buf.write("-- %s\n" % inner_name)
    buf.write("-- game_items 按部署资源(resources/modern/share/item 的 tool_item.ini\n")
    buf.write("-- + equipment.ini,归类与取值口径同 0002)刷新,幂等 REPLACE 可重放。\n")
    buf.write("-- 共 %d 行。生成工具:tools/game_items_refresh.py。\n\n" % len(rows))
    for i in range(0, len(rows), chunk):
        buf.write("REPLACE INTO `game_items` VALUES\n")
        buf.write(",\n".join(tuple_sql(r) for r in rows[i:i + chunk]))
        buf.write(";\n\n")
    sql = buf.getvalue().encode("utf-8")
    tf = tarfile.open(out_path, "w:gz", format=tarfile.USTAR_FORMAT)
    info = tarfile.TarInfo(inner_name)
    info.size, info.mode, info.mtime = len(sql), 0o644, 0
    tf.addfile(info, io.BytesIO(sql))
    tf.close()
    return os.path.getsize(out_path)


def load_idres_names(path=F_IDRES):
    """stringname.idres → {小写键: 值}；键重复取首值、空值跳过
    （对齐 Java ResourceCache.idresLowerCased）。"""
    names = {}
    for raw in open(path, encoding="utf-8-sig"):
        raw = raw.strip()
        if "=" not in raw:
            continue
        k, v = raw.split("=", 1)
        k, v = k.strip().lower(), v.strip()
        if k and v and k not in names:
            names[k] = v
    return names


def placeholder_name(name, item_id):
    return name == "" or name.isdigit() or name == item_id


def db_connect():
    import pymysql
    dsn = os.environ.get("NINEYIN_MYSQL_DSN",
                         "nineyin:nineyin123@tcp(127.0.0.1:3306)/nineyin")
    m = re.match(r"(\w+):([^@]*)@tcp\(([^):]+)(?::(\d+))?\)/(\w+)", dsn)
    if not m:
        raise SystemExit("无法解析 NINEYIN_MYSQL_DSN: %s" % dsn)
    user, pwd, host, port, db = m.group(1), m.group(2), m.group(3), int(m.group(4) or 3306), m.group(5)
    return pymysql.connect(host=host, port=port, user=user, password=pwd,
                           database=db, charset="utf8mb4")


def gen_fix_names(out_path, chunk=500):
    """占位名行按 idres 修真名；全行取自活库仅改 item_name。"""
    names = load_idres_names()
    conn = db_connect()
    cur = conn.cursor()
    cur.execute("SELECT %s FROM game_items" % ", ".join("`%s`" % c for c in COLS))
    fixed = []
    for row in cur.fetchall():
        row = list(row)
        if not placeholder_name(row[2], row[0]):
            continue
        real = names.get(row[0].lower())
        if not real or real == row[2]:
            continue
        row[2] = real
        fixed.append(row)
    conn.close()
    fixed.sort(key=lambda r: r[0])
    inner = os.path.basename(out_path).replace(".sql.gz", ".sql")
    buf = io.StringIO()
    buf.write("-- %s\n" % inner)
    buf.write("-- game_items 占位名(纯数字编号/空/==item_id,共 %d 行)按\n" % len(fixed))
    buf.write("-- stringname.idres 修真名,口径同运行时 gmItemDisplayName(idres 优先)。\n")
    buf.write("-- 全行取自活库仅改 item_name,其余列零扰动;幂等 REPLACE 可重放。\n\n")
    for i in range(0, len(fixed), chunk):
        buf.write("REPLACE INTO `game_items` VALUES\n")
        buf.write(",\n".join(tuple_sql(r) for r in fixed[i:i + chunk]))
        buf.write(";\n\n")
    sql = buf.getvalue().encode("utf-8")
    tf = tarfile.open(out_path, "w:gz", format=tarfile.USTAR_FORMAT)
    info = tarfile.TarInfo(inner)
    info.size, info.mode, info.mtime = len(sql), 0o644, 0
    tf.addfile(info, io.BytesIO(sql))
    tf.close()
    print("written %s (%d bytes, %d rows)" % (out_path, os.path.getsize(out_path), len(fixed)))
    return len(fixed)


def main():
    if len(sys.argv) >= 3 and sys.argv[1] == "--fix-names":
        gen_fix_names(sys.argv[2])
        return
    if len(sys.argv) >= 3 and sys.argv[1] == "--write":
        added, changed, _, _, _ = compute_refresh()
        rows = sorted(added + changed, key=lambda r: r[0])
        size = write_migration(rows, sys.argv[2],
                               os.path.basename(sys.argv[2]).replace(".sql.gz", ".sql"))
        print("written %s (%d bytes, %d rows)" % (sys.argv[2], size, len(rows)))
        return
    added, changed, artifacts, removed, base = compute_refresh()
    print("基线 %d 行 / 新增 %d / 真实变更 %d / 解码伪影跳过 %d / 移除 %d" % (
        base, len(added), len(changed), len(artifacts), len(removed)))
    for r in changed:
        print("  变更 %s: func_pack -> %s (%s)" % (r[0], r[8], r[2]))
    if artifacts:
        print("  伪影:", ", ".join(artifacts))
    if removed:
        print("  移除:", ", ".join(removed[:20]))


if __name__ == "__main__":
    main()
