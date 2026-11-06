# sql/ — 数据库结构与配置数据权威存档

本目录是 **nineyin 库（F:\Game\Online\9yin\9yin-server-go\mysql, 127.0.0.1:3306）结构与配置数据的唯一权威版本存档**。
数据库升级（含新库重建、缺表补建、配置恢复）统一通过本目录完成——`deploy.bat` 第 [3] 步即调用 `upgrade-db.bat` 重放本目录全部脚本。

## 文件格式

- `.sql.gz` = **tar+gzip 单文件归档**（Windows 自带 `tar.exe` 可直接读写，不依赖外部 gzip）
  - 查看内容：`tar -xOzf 0001_schema_20261031.sql.gz`
  - 解压还原：`tar -xzf xxx.sql.gz -C 目录`
- 压缩包内部文件名与压缩包同名（`.sql` 后缀）

## 命名规则

`NNNN_名称_YYYYMMDD.sql.gz`，执行时**按文件名序**依次重放：

| 文件 | 内容 |
|---|---|
| `0001_schema_20261031.sql.gz` | 全部 67 张表结构（`CREATE TABLE IF NOT EXISTS`，无 DROP，可重放） |
| `0002_config_data_20261031.sql.gz` | 26 张配置表数据（`game_*` / `server_*` / `teams` / `team_members` / `gm_permissions` / `schema_migrations`，`REPLACE INTO`） |
| `NNNN_其它` | 后续增量（新表 / 加字段 / 数据修正） |

**边界**：角色运行时表（`accounts` / `roles` / `role_*` 共 41 张）只存**结构**不存数据——角色数据以运行库为准，不进存档。

## 日常操作

```bat
rem 升级数据库（deploy.bat 自动调用；也可独立跑）
upgrade-db.bat [库名]          rem 缺省 nineyin；指定库名可在测试库上验证重建

rem 加了表 / 加了字段后，导出该表存档（结构+数据单文件）
export-sql.bat --table "新表名" --name add_xxx
export-sql.bat --table "t1,t2" --name fix_yyy     rem 多表逗号分隔（cmd 下注意引号）

rem 全量重导基线（结构+配置数据，产生新序号的两个文件；仅在基线大面积变化时用）
export-sql.bat
```

## 增量脚本编写要求

每个文件都必须**幂等**（可重复执行、不动已有数据）：

- 建表：`CREATE TABLE IF NOT EXISTS`
- 加列（MySQL 无 ADD COLUMN IF NOT EXISTS，用 information_schema 条件判断，
  参照 `tools/data-import/src/main/resources/schema/0003_server_list_and_terms.sql`）
- 数据：`REPLACE INTO`（以存档为准）或 `INSERT ... WHERE NOT EXISTS` 引导

手写增量 SQL 的打包方式（文件名与包内名一致）：

```bat
tar -czf sql\0003_add_xxx_20261101.sql.gz -C 存放目录 0003_add_xxx_20261101.sql
```

## 相关脚本（仓库根）

- `export-sql.bat` — 从运行库导出存档（本文档"日常操作"）
- `upgrade-db.bat` — 按序重放本目录全部 `.sql.gz`（跳过非 `NNNN_` 前缀的文件）
- `deploy.bat` — 完整部署一条龙，第 [3] 步调用 `upgrade-db.bat`
- 导出/升级均用 `nineyin` 账号（仅 `nineyin` 库权限，无建库权限；
  新建库需先用 root：`CREATE DATABASE xxx CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`）
