-- 给 nineyin 分配官方 dump 三库权限：nx_base / nx_district / nx_log。
-- root 执行：mysql -h127.0.0.1 -P3306 -uroot -p < grant_nineyin.sql
-- 幂等：账号已存在不动；已授权重复执行无副作用（GRANT 幂等）。
-- 口令对齐 env.server 的 NINEYIN_MYSQL_DSN（nineyin:nineyin123），
-- CREATE USER IF NOT EXISTS 不改已有密码，显式重置兜底。

-- 本机 DSN 走 tcp(127.0.0.1:3306)，localhost/127.0.0.1 两主机够用；
-- 换机器/远程连接时再启用 % 行（慎重：等于放开远程口令面）
CREATE USER IF NOT EXISTS 'nineyin'@'localhost' IDENTIFIED BY 'nineyin123';
CREATE USER IF NOT EXISTS 'nineyin'@'127.0.0.1' IDENTIFIED BY 'nineyin123';
-- CREATE USER IF NOT EXISTS 'nineyin'@'%' IDENTIFIED BY 'nineyin123';

ALTER USER 'nineyin'@'localhost' IDENTIFIED BY 'nineyin123';
ALTER USER 'nineyin'@'127.0.0.1' IDENTIFIED BY 'nineyin123';

-- 逐库授权：官方 dump 导入目标库（DDL 见同目录 nx_base.sql 等），
-- 日常读写/建表均够用；nineyin 自有库权限不在本文件范围
GRANT ALL PRIVILEGES ON nx_base.*     TO 'nineyin'@'localhost', 'nineyin'@'127.0.0.1';
GRANT ALL PRIVILEGES ON nx_district.* TO 'nineyin'@'localhost', 'nineyin'@'127.0.0.1';
GRANT ALL PRIVILEGES ON nx_log.*      TO 'nineyin'@'localhost', 'nineyin'@'127.0.0.1';

FLUSH PRIVILEGES;
