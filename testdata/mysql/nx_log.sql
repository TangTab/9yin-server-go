DROP TABLE IF EXISTS `nx_log`.`deed_log`;
CREATE TABLE `nx_log`.`deed_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `d_create` datetime DEFAULT NULL,
  `s_act_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `n_type` int(11) DEFAULT NULL,
  `n_act_value` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `lx_deed_log2` (`role`),
  KEY `lx_deed_log1` (`d_create`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`instance_npc_log`;
CREATE TABLE `nx_log`.`instance_npc_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `npcname` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `sceneid` int(11) DEFAULT NULL,
  `location` varchar(32) DEFAULT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_instance_npc_log1` (`time`),
  KEY `ix_instance_npc_log2` (`npcname`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`nx_chatlog`;
CREATE TABLE `nx_log`.`nx_chatlog` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `chat_time` datetime DEFAULT NULL,
  `chat_from` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `chat_to` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `chat_type` int(11) DEFAULT NULL,
  `chat_content` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `chat_comment` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`chat_time`),
  KEY `IX_nx_chatlog_2` (`chat_from`),
  KEY `IX_nx_chatlog_3` (`chat_to`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`nx_gmlog`;
CREATE TABLE `nx_log`.`nx_gmlog` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `log_time` datetime DEFAULT NULL,
  `log_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_source` int(11) DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `log_content` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_comment` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_gmlog_1` (`log_time`),
  KEY `IX_nx_gmlog_2` (`log_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`nx_itemlog`;
CREATE TABLE `nx_log`.`nx_itemlog` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `item_id` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_user` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_op` int(11) DEFAULT NULL,
  `item_reason` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_time` datetime DEFAULT NULL,
  `item_comment` varchar(2048) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_itemlog_1` (`item_id`),
  KEY `IX_nx_itemlog_2` (`item_name`),
  KEY `IX_nx_itemlog_3` (`item_user`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`nx_logdata`;
CREATE TABLE `nx_log`.`nx_logdata` (
  `serial_no` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `log_time` datetime DEFAULT NULL,
  `log_name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_source` int(11) DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `log_content` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_comment` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`player_exp_log`;
CREATE TABLE `nx_log`.`player_exp_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `logtype` int(11) DEFAULT NULL,
  `amount` int(11) DEFAULT NULL,
  `comment` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_player_exp_log1` (`time`),
  KEY `ix_player_exp_log2` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_cre_del_log`;
CREATE TABLE `nx_log`.`role_cre_del_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`),
  KEY `IX_nx_chatlog_4` (`account`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_log`.`role_experience_log`;
CREATE TABLE `nx_log`.`role_experience_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `add_exp` int(11) DEFAULT NULL,
  `cur_level` int(11) DEFAULT NULL,
  `cur_exp` int(11) DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_log`.`role_guild_log`;
CREATE TABLE `nx_log`.`role_guild_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(4) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `target_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `guild_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_player_exp_log1` (`time`),
  KEY `ix_player_exp_log2` (`log_type`),
  KEY `ix_player_exp_log3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_item_change_log`;
CREATE TABLE `nx_log`.`role_item_change_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `item_config_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_unique_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `success` int(11) DEFAULT NULL,
  `consume_material` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_login_log`;
CREATE TABLE `nx_log`.`role_login_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `entry_time` datetime DEFAULT NULL,
  `duration` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_scene_log`;
CREATE TABLE `nx_log`.`role_scene_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `entry_time` datetime DEFAULT NULL,
  `duration` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_task_log`;
CREATE TABLE `nx_log`.`role_task_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `task_id` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_transmit_log`;
CREATE TABLE `nx_log`.`role_transmit_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `target_scene_id` int(11) DEFAULT NULL,
  `target_x` float(16,4) DEFAULT NULL,
  `target_y` float(16,4) DEFAULT NULL,
  `target_z` float(16,4) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`total_capital_log`;
CREATE TABLE `nx_log`.`total_capital_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `res` int(11) DEFAULT NULL,
  `capital_type` int(11) DEFAULT NULL,
  `inc_value` decimal(20,0) DEFAULT NULL,
  `old_value` decimal(20,0) DEFAULT NULL,
  `new_value` decimal(20,0) DEFAULT NULL,
  `gm` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(255) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_item_log`;
CREATE TABLE `nx_log`.`role_item_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `item_config_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_unique_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `amount` bigint(20) DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`),
  KEY `IX_nx_chatlog_4` (`item_config_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_enjoy_log`;
CREATE TABLE `nx_log`.`role_enjoy_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(4) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `step_id` int(4) DEFAULT NULL,
  `act_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `job` int(11) DEFAULT NULL,
  `level` int(11) DEFAULT NULL,
  `guild_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_player_exp_log1` (`time`),
  KEY `ix_player_exp_log2` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`deal_log`;
CREATE TABLE `nx_log`.`deal_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `tradeid` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `buyer` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `seller` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `itemid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `instance_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `count` int(11) DEFAULT NULL,
  `getorlost` int(11) DEFAULT NULL,
  `tradechannel` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_deal_log1` (`time`),
  KEY `ix_deal_log2` (`buyer`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`game_player_action_log`;
CREATE TABLE `nx_log`.`game_player_action_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `s_account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `s_role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `d_create` datetime DEFAULT NULL,
  `s_act_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `n_type` int(11) DEFAULT NULL,
  `n_act_value` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `nx_log`.`instance_log`;
CREATE TABLE `nx_log`.`instance_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `instanceid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `instancetypeid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `starttime` datetime DEFAULT NULL,
  `endtime` datetime DEFAULT NULL,
  `duration` int(11) DEFAULT NULL,
  `troop_count1` int(11) DEFAULT NULL,
  `troop_count2` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_instance_log1` (`starttime`),
  KEY `ix_instance_log2` (`instancetypeid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`instance_role_log`;
CREATE TABLE `nx_log`.`instance_role_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `instanceid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `instancetypeid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `entertime` datetime DEFAULT NULL,
  `leavetime` datetime DEFAULT NULL,
  `duration` int(11) DEFAULT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_instance_role_log1` (`entertime`),
  KEY `ix_instance_role_log2` (`instancetypeid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`item_log`;
CREATE TABLE `nx_log`.`item_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `itemfrom` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `itemto` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `itemid` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `instance_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_num` bigint(20) DEFAULT NULL,
  `logtype` int(11) DEFAULT NULL,
  `comment1` varchar(255) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment2` varchar(255) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `comment3` varchar(255) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_item_log1` (`time`),
  KEY `ix_item_log2` (`itemfrom`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`program_log`;
CREATE TABLE `nx_log`.`program_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `programid` int(11) DEFAULT NULL,
  `programname` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `opentime` datetime DEFAULT NULL,
  `endtime` datetime DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_program_log1` (`opentime`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`punish_log`;
CREATE TABLE `nx_log`.`punish_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `operator` varchar(100) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `minutes` int(11) DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_punish_log1` (`time`),
  KEY `ix_punish_log2` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_log`;
CREATE TABLE `nx_log`.`role_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `account` varchar(100) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `logintime` datetime DEFAULT NULL,
  `logouttime` datetime DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `lx_role_log1` (`logintime`),
  KEY `lx_role_log2` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_upgrade_log`;
CREATE TABLE `nx_log`.`role_upgrade_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `logtype` int(11) DEFAULT NULL,
  `level_before` int(11) DEFAULT NULL,
  `level_after` int(11) DEFAULT NULL,
  `skill_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `account` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`scene_log`;
CREATE TABLE `nx_log`.`scene_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `sceneid` int(11) DEFAULT NULL,
  `date` datetime DEFAULT NULL,
  `enter` int(11) DEFAULT NULL,
  `totaltime` decimal(8,2) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_scene_log1` (`date`) USING BTREE,
  KEY `ix_scene_log2` (`sceneid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`task_log`;
CREATE TABLE `nx_log`.`task_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `role` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `task` int(11) DEFAULT NULL,
  `logtype` int(11) DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_task_log1` (`time`),
  KEY `ix_task_log2` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_illegal_req_log`;
CREATE TABLE `nx_log`.`role_illegal_req_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `camp` int(11) DEFAULT NULL,
  `job` int(11) DEFAULT NULL,
  `level` int(11) DEFAULT NULL,
  `msg_id` int(11) DEFAULT NULL,
  `sub_msg_id` int(11) DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_player_exp_log1` (`time`),
  KEY `ix_player_exp_log2` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`sys_total_capital_snap`;
CREATE TABLE `nx_log`.`sys_total_capital_snap` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `tcrc_type` int(11) DEFAULT NULL,
  `tcrc_switch` int(11) DEFAULT NULL,
  `tcrc_sys_total` bigint(20) DEFAULT NULL,
  `tcrc_can_buy` bigint(20) DEFAULT NULL,
  `tcrc_can_prize` bigint(20) DEFAULT NULL,
  `tcrc_user_cur` bigint(20) DEFAULT NULL,
  `tcrc_offline_prize` bigint(20) DEFAULT NULL,
  `tcrc_user_onway` bigint(20) DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `ix_player_exp_log1` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`role_auction_log`;
CREATE TABLE `nx_log`.`role_auction_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `log_type` int(11) DEFAULT NULL,
  `ip` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `mac` varchar(16) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_scene_id` int(11) DEFAULT NULL,
  `cur_x` float(16,4) DEFAULT NULL,
  `cur_y` float(16,4) DEFAULT NULL,
  `cur_z` float(16,4) DEFAULT NULL,
  `item_config_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `item_unique_id` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `amount` int(11) DEFAULT NULL,
  `money` int(11) DEFAULT NULL,
  `comment` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `IX_nx_chatlog_1` (`time`),
  KEY `IX_nx_chatlog_2` (`log_type`),
  KEY `IX_nx_chatlog_3` (`role_name`),
  KEY `IX_nx_chatlog_4` (`item_config_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;

DROP TABLE IF EXISTS `nx_log`.`sys_rank_log`;
CREATE TABLE `nx_log`.`sys_rank_log` (
  `serial` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `time` datetime DEFAULT NULL,
  `rank_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `cur_No` int(11) DEFAULT NULL,
  `last_No` int(11) DEFAULT NULL,
  `role_uid` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `role_name` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `main_sort` bigint(20) DEFAULT NULL,
  `sub_sort` bigint(20) DEFAULT NULL,
  `camp_or_job` int(11) DEFAULT NULL,
  `guild` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;
