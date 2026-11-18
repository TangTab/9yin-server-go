DROP TABLE IF EXISTS `nx_district`.`dc_accounts`;
CREATE TABLE `nx_district`.`dc_accounts` (
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `password` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `lastlog_time` datetime DEFAULT NULL,
  `lastlog_addr` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `valid_time` datetime DEFAULT NULL,
  `user_type` int(11) DEFAULT NULL,
  `status` int(11) DEFAULT NULL,
  `logged` int(11) DEFAULT NULL,
  `charge_mode` int(11) DEFAULT NULL,
  `points` int(11) DEFAULT NULL,
  `used_points` int(11) DEFAULT NULL,
  `comment` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;
