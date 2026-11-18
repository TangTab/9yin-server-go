DROP TABLE IF EXISTS `nx_base`.`game_enum`;
CREATE TABLE `nx_base`.`game_enum` (
  `name` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `value` int(11) NOT NULL,
  `desc` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`name`,`value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`log_enumtype`;
CREATE TABLE `nx_base`.`log_enumtype` (
  `type` int(11) NOT NULL DEFAULT '0',
  `desc` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`log_enumvalue`;
CREATE TABLE `nx_base`.`log_enumvalue` (
  `type` int(11) NOT NULL,
  `value` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `text` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `desc` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`value`,`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`log_type`;
CREATE TABLE `nx_base`.`log_type` (
  `log_type` int(11) NOT NULL,
  `desc` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`log_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_base`.`nx_auction`;
CREATE TABLE `nx_base`.`nx_auction` (
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `create_time` datetime DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `record` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `version` int(11) DEFAULT NULL,
  `amount` int(11) DEFAULT NULL,
  `range` int(11) DEFAULT NULL,
  `save_time` datetime DEFAULT NULL,
  `save_data` longblob,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_base`.`nx_chunks`;
CREATE TABLE `nx_base`.`nx_chunks` (
  `serial_no` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `server_id` int(11) DEFAULT NULL,
  `scene_id` int(11) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `deleted` int(11) DEFAULT NULL,
  `save_time` datetime DEFAULT NULL,
  `save_data` longblob,
  PRIMARY KEY (`serial_no`),
  KEY `IX_nx_chunks_1` (`name`),
  KEY `IX_nx_chunks_2` (`scene_id`),
  KEY `IX_nx_chunks_3` (`server_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_config`;
CREATE TABLE `nx_base`.`nx_config` (
  `id` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `property` longtext CHARACTER SET utf8 COLLATE utf8_bin,
  `status` int(11) DEFAULT NULL,
  `op_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `op_type` int(11) DEFAULT NULL,
  `op_property` longtext CHARACTER SET utf8 COLLATE utf8_bin,
  `op_status` int(11) DEFAULT NULL,
  `op_version` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `IX_nx_config_1` (`name`),
  KEY `IX_nx_config_2` (`op_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_domains`;
CREATE TABLE `nx_base`.`nx_domains` (
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `create_time` datetime DEFAULT NULL,
  `deleted` int(11) DEFAULT NULL,
  `save_time` datetime DEFAULT NULL,
  `save_data` longblob,
  PRIMARY KEY (`name`),
  UNIQUE KEY `IX_nx_domains` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_guilds`;
CREATE TABLE `nx_base`.`nx_guilds` (
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `create_time` datetime DEFAULT NULL,
  `deleted` int(11) DEFAULT NULL,
  `save_time` datetime DEFAULT NULL,
  `save_data` longblob,
  PRIMARY KEY (`name`),
  UNIQUE KEY `IX_nx_guilds` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_letter`;
CREATE TABLE `nx_base`.`nx_letter` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `msg_send_time` datetime DEFAULT NULL,
  `msg_time` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `msg_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `msg_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `msg_source` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `msg_source_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `msg_type` int(11) DEFAULT NULL,
  `msg_content` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `msg_appendix` varchar(4096) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`serial_no`),
  KEY `IX_nx_letter_1` (`msg_send_time`),
  KEY `IX_nx_letter_2` (`msg_name`),
  KEY `IX_nx_letter_3` (`msg_uid`),
  KEY `IX_nx_letter_4` (`msg_source`),
  KEY `IX_nx_letter_5` (`msg_source_uid`),
  KEY `IX_nx_letter_6` (`msg_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_pubdata`;
CREATE TABLE `nx_base`.`nx_pubdata` (
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `create_time` datetime DEFAULT NULL,
  `deleted` int(11) DEFAULT NULL,
  `save_time` datetime DEFAULT NULL,
  `save_data` longblob,
  PRIMARY KEY (`name`),
  UNIQUE KEY `IX_nx_pubdata` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_roles`;
CREATE TABLE `nx_base`.`nx_roles` (
  `name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `account` varchar(128) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
  `create_time` datetime DEFAULT NULL,
  `delete_time` datetime DEFAULT NULL,
  `open_time` datetime DEFAULT NULL,
  `server_id` int(11) DEFAULT NULL,
  `total_sec` int(11) DEFAULT NULL,
  `deleted` int(11) DEFAULT NULL,
  `letter_max` int(11) DEFAULT NULL,
  `sys_flags` int(11) DEFAULT NULL,
  `illegals` int(11) DEFAULT NULL,
  `online` int(11) DEFAULT NULL,
  `role_index` int(11) DEFAULT NULL,
  `scene_id` int(11) DEFAULT NULL,
  `location` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `resume_time` datetime DEFAULT NULL,
  `resume_info` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `charge_info` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `save_para1` varchar(2048) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `save_time` datetime DEFAULT NULL,
  `save_data` longblob,
  `lazy_data` longblob,
  PRIMARY KEY (`name`),
  UNIQUE KEY `IX_nx_roles` (`uid`),
  KEY `IX_nx_roles_1` (`account`),
  KEY `IX_nx_roles_2` (`save_time`),
  KEY `IX_nx_roles_3` (`resume_time`),
  KEY `IX_nx_roles_4` (`server_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_role_dumpinfo`;
CREATE TABLE `nx_base`.`nx_role_dumpinfo` (
  `role_name` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `role_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_account` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `delete_time` datetime DEFAULT NULL,
  `deleted` int(11) DEFAULT NULL,
  `total_sec` int(11) DEFAULT NULL,
  `sex` int(11) DEFAULT NULL,
  `camp` int(11) DEFAULT NULL,
  `race` int(11) DEFAULT NULL,
  `job` int(11) DEFAULT NULL,
  `level` int(11) DEFAULT NULL,
  `fight_power` int(11) DEFAULT NULL,
  `money_rec` longtext CHARACTER SET utf8 COLLATE utf8_bin,
  PRIMARY KEY (`role_name`),
  KEY `IX_nx_role_dumpinfo_0` (`role_uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_sns_feed`;
CREATE TABLE `nx_base`.`nx_sns_feed` (
  `feed_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `target_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `range_mask` int(11) DEFAULT NULL,
  `level` int(11) DEFAULT NULL,
  `category` int(11) DEFAULT NULL,
  `desc_info` varchar(1024) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment_at` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `good_count` int(11) DEFAULT NULL,
  `bad_count` int(11) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  PRIMARY KEY (`feed_id`),
  KEY `IX_nx_sns_feed_1` (`uid`),
  KEY `IX_nx_sns_feed_2` (`target_uid`),
  KEY `IX_nx_sns_feed_3` (`range_mask`),
  KEY `IX_nx_sns_feed_4` (`level`),
  KEY `IX_nx_sns_feed_5` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_sns_relation`;
CREATE TABLE `nx_base`.`nx_sns_relation` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `target_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `range_mask` int(11) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  PRIMARY KEY (`serial_no`),
  KEY `IX_nx_sns_relation_1` (`uid`),
  KEY `IX_nx_sns_relation_2` (`target_uid`),
  KEY `IX_nx_sns_relation_3` (`range_mask`),
  KEY `IX_nx_sns_relation_4` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_base`.`nx_trade`;
CREATE TABLE `nx_base`.`nx_trade` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `seller_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `seller_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `buyer_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `buyer_uid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `buyer_count` int(11) DEFAULT NULL,
  `item_type` int(11) DEFAULT NULL,
  `item_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_info` varchar(2048) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_cat1` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_cat2` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_cat3` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `status` int(11) DEFAULT NULL,
  `price_min` int(11) DEFAULT NULL,
  `price_max` int(11) DEFAULT NULL,
  `price_deal` int(11) DEFAULT NULL,
  `time_beg` datetime DEFAULT NULL,
  `time_end` datetime DEFAULT NULL,
  `time_deal` datetime DEFAULT NULL,
  PRIMARY KEY (`serial_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
