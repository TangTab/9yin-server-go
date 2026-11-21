-- 建 sa 账号与基础库：官方 ini ConnectString（User ID=sa;Password=abc&4621）
-- 的数据库侧前置。root 执行：mysql -u root -p < create_sa_account.sql
-- 幂等：库已存在不动；账号已存在则统一重置密码。
-- 官方 dump（nx_base.sql 等）DEFINER=`sa`@`%`，故 sa@% 必建；
-- 1045 报 'sa'@'localhost'，故 localhost/127.0.0.1 一并建齐。

-- nx_base1/nx_district1：单机 game_unite.ini [Role]/[Account] 的 Initial Catalog
-- nx_base/nx_district/nx_log：官方 dump 导入目标库
CREATE DATABASE IF NOT EXISTS nx_base1;
CREATE DATABASE IF NOT EXISTS nx_district1;
CREATE DATABASE IF NOT EXISTS nx_base;
CREATE DATABASE IF NOT EXISTS nx_district;
CREATE DATABASE IF NOT EXISTS nx_log;

CREATE USER IF NOT EXISTS 'sa'@'localhost' IDENTIFIED BY 'abc&4621';
CREATE USER IF NOT EXISTS 'sa'@'127.0.0.1' IDENTIFIED BY 'abc&4621';
CREATE USER IF NOT EXISTS 'sa'@'%'         IDENTIFIED BY 'abc&4621';

-- CREATE USER IF NOT EXISTS 不改已有密码，显式重置兜底
ALTER USER 'sa'@'localhost' IDENTIFIED BY 'abc&4621';
ALTER USER 'sa'@'127.0.0.1' IDENTIFIED BY 'abc&4621';
ALTER USER 'sa'@'%'         IDENTIFIED BY 'abc&4621';

-- 逐库授权：日常读写/建表/导入官方 dump 均够用（ensureDcAccounts 的
-- CREATE DATABASE 已由上面预建库消解，无需全局权限）
GRANT ALL PRIVILEGES ON nx_base1.*     TO 'sa'@'localhost', 'sa'@'127.0.0.1', 'sa'@'%';
GRANT ALL PRIVILEGES ON nx_district1.* TO 'sa'@'localhost', 'sa'@'127.0.0.1', 'sa'@'%';
GRANT ALL PRIVILEGES ON nx_base.*      TO 'sa'@'localhost', 'sa'@'127.0.0.1', 'sa'@'%';
GRANT ALL PRIVILEGES ON nx_district.*  TO 'sa'@'localhost', 'sa'@'127.0.0.1', 'sa'@'%';
GRANT ALL PRIVILEGES ON nx_log.*       TO 'sa'@'localhost', 'sa'@'127.0.0.1', 'sa'@'%';

-- 仅当需要运行时建库（手动删库后重跑等）或以 sa 自导入含 DEFINER 的
-- dump 时，再启用全局行：
-- GRANT ALL PRIVILEGES ON *.* TO 'sa'@'localhost', 'sa'@'127.0.0.1', 'sa'@'%' WITH GRANT OPTION;

FLUSH PRIVILEGES;
