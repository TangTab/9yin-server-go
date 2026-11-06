# 九阴客户端 lua64 协议字典（生成物）

> 由 `tools/lua64_extract.py` 从客户端解包脚本 `lua64_unpacked/lua64` 自动提取。
> 类型标记: int/str/wstr(UTF16串)/flt/obj(场景对象ident)/i64(金额文)/... (变参)。
> 服务端→客户端为**命令名寻址**：引擎按消息名调用 custom_handler.lua 同名函数，
> 其 msg_type 之后的形参即下行载荷布局（见第三节）。

## 一、客户端→服务端 custom 消息（按 ID）

| ID | 常量名 | 客户端发送函数(参数类型序列) |
|---|---|---|
| 2 | CLIENT_CUSTOMMSG_GM | `custom_gminfo(var)` |
| 3 | CLIENT_CUSTOMMSG_MOVE_TYPE | `custom_change_move_type(int)` |
| 4 | CLIENT_CUSTOMMSG_TEST_SERVER | `custom_ws_check_test_server_open(int)` |
| 5 | CLIENT_CUSTOMMSG_ANSWERMESSAGEBOX | `custom_send_msgbox_results_msg(int,int,...)` |
| 6 | CLIENT_CUSTOMMSG_ANSWERINPUTBOX | `custom_send_inputbox_results_msg(int,wstr)` |
| 8 | CLIENT_CUSTOMMSG_GUILD_WAR | `custom_occupy_item_guild_war_info(int)`; `custom_use_guild_war_occupy_item(int,int,int,int,int)`; `send_faction_msg(int,int,int)` |
| 16 | CLIENT_CUSTOMMSG_CROSS_SERVER | `custom_cross_server(int,...)` |
| 17 | CLIENT_CUSTOMMSG_NEW_COMPOSITE | `custom_new_composite(int,...)` |
| 19 | CLIENT_CUSTOMMSG_FASHION_SUIT | `custom_send_fashion_suit(int,...)` |
| 32 | CLIENT_CUSTOMMSG_CONVERT_ITEM | `custom_convert_item(int,...)` |
| 34 | CLIENT_CUSTOMMSG_DELETEITEM | `custom_delete_item(int,int,int)` |
| 36 | CLIENT_CUSTOMMSG_ARANGEITEM | `custom_arrange_item(int,int,int)`; `custom_arrange_item_by_type(int,int,int,int)` |
| 38 | CLIENT_CUSTOMMSG_REPAIR_ADDBAG | `custom_send_repair_addbags(int,int,int)` |
| 39 | CLIENT_CUSTOMMSG_BINDITEM | `custom_bind_item(int,int)` |
| 40 | CLIENT_CUSTOMMSG_REPAIRITEM | `custom_send_repair_item(int,int,int,int,int)` |
| 41 | CLIENT_CUSTOMMSG_REPAIRALL | `custom_send_repair_all(int,int,int)` |
| 42 | CLIENT_CUSTOMMSG_PROTECTITEMADD | `custom_add_protect_item(int,int)` |
| 43 | CLIENT_CUSTOMMSG_PROTECTITEMDEL | `custom_del_protect_item(str)` |
| 50 | CLIENT_CUSTOMMSG_SELECT_BIND | `custom_select_bind(var,var,var)` |
| 52 | CLIENT_CUSTOMMSG_FURNITURE | `custom_home_furn(...)` |
| 53 | CLIENT_CUSTOMMSG_MOVEITEM_TO_NEW | `custom_send_move_item_to_new(int,str)` |
| 61 | CLIENT_CUSTOMMSG_SHOWEQUIP_TYPE | `custom_showequip_type(int)` |
| 63 | CLIENT_CUSTOMMSG_REPLY_TRANSFER_SERVER | `custom_reply_trans_server_list(int,...)` |
| 64 | CLIENT_CUSTOMMSG_REQUEST_SHOP_EXCHANGE_FORM | `custom_request_shop_exchange_form(int,int,str,int,int)` |
| 65 | CLIENT_CUSTOMMSG_REQUEST_SWAP_GOODS | `custom_request_swap_goods(int,...)` |
| 69 | CLIENT_CUSTOMMSG_REQUEST_CONDITION_SHOP | `custom_condition_details(int)` |
| 70 | CLIENT_CUSTOMMSG_BUY_FROM_SHOP | `custom_buy_item(str,int,int,int)` |
| 71 | CLIENT_CUSTOMMSG_SELL_TO_SHOP | `custom_sell_item(int,int,int,str)` |
| 73 | CLIENT_CUSTOMMSG_BUY_BACK_FORM_SHOP | `custom_buy_back(var)` |
| 79 | CLIENT_CUSTOMMSG_EXCHANGE_FROM_SHOP | `custom_exchange_item(str,int,int,int)` |
| 80 | CLIENT_CUSTOMMSG_GIVE_ITEM_MSG_TYPE | `custom_give_item(int,int,str)` |
| 81 | CLIENT_CUSTOMMSG_CAPITAL_GIRD | `custom_send_open_capital(int)` |
| 83 | CLIENT_CUSTOMMSG_BATTLEFIELD | `custom_battlefield(int,...)` |
| 84 | CLIENT_CUSTOMMSG_RANDOM_CLONE_SCENE | `custom_random_clone(int,...)` |
| 99 | CLIENT_CUSTOMMSG_SET_OPERATEMODE | `custom_set_operate_mode(int)` |
| 100 | CLIENT_CUSTOMMSG_REQUEST | `custom_request(int,var,...)`; `custom_request_teacher(int,wstr)`; `custom_protocolfreerelation(int,wstr)` |
| 101 | CLIENT_CUSTOMMSG_REQUEST_ANSWER | `custom_request_answer(int,var,int)` |
| 108 | CLIENT_CUSTOMMSG_GET_SCHOOLLEADER_INFO | `custom_get_schoolleader_info()` |
| 110 | CLIENT_CUSTOMMSG_XLXF_MSG | `custom_xialuxiangfeng(int,...)` |
| 111 | CLIENT_CUSTOMMSG_ITEM_EXCHANGE | `custom_item_exchange(str,int,str)` |
| 112 | CLIENT_CUSTOMMSG_ENTRY_CLONE | `custom_entry_clone(var,var)` |
| 116 | CLIENT_CUSTOMMSG_REMARK_NAME | `custom_remark_name_oper(int,str,wstr)` |
| 119 | CLIENT_CUSTOMMSG_CHITCHAT_REQUEST_CHAT | `custom_request_chat(wstr)` |
| 120 | CLIENT_CUSTOMMSG_RELATION | `custom_add_relation(int,wstr,int,int)` |
| 121 | CLIENT_CUSTOMMSG_FEED | `custom_sync_recent_feeds(int)`; `custom_sync_recent_feeds(int,wstr,str,wstr)`; `custom_sync_recent_feeds(int,str)`; `custom_sync_recent_feeds(int,str)` +4 |
| 123 | CLIENT_CUSTOMMSG_SNS_SEARCH_FRIEND | `custom_sns_search_friend(wstr)` |
| 129 | CLIENT_CUSTOMMSG_FEED_BACK | `custom_sns_feed_back_goto(int,wstr)`; `custom_sns_feed_back_broadcast(int,wstr)` |
| 130 | CLIENT_CUSTOMMSG_RESET_ALL_CLONE | `reset_all_clone()` |
| 131 | CLIENT_CUSTOMMSG_RESET_SAVE_CLONE | `reset_save_clone(str,int)` |
| 132 | CLIENT_CUSTOMMSG_CHITCHAT_SEND_GROUP_CONTENT | `custom_send_group_chat_content(wstr,str)` |
| 133 | CLIENT_CUSTOMMSG_SELL_ITEM_TOWEB | `custom_send_webexchange_msg(int,...)` |
| 134 | CLIENT_CUSTOMMSG_CAPTAIN_RESET_SAVE_CLONE | `captain_reset_save_clone(str,int)` |
| 135 | CLIENT_CUSTOMMSG_INVITE_HELPER | `custom_avatar_clone_invite_help(wstr)` |
| 140 | CLIENT_CUSTOMMSG_GET_LEAVE_TIME_CLONE | `get_leave_time_clone()` |
| 142 | CLIENT_CUSTOMMSG_CLONE_CHALLENGE_COUNT | `custom_send_clone_challenge_rec(obj)` |
| 143 | CLIENT_CUSTOMMSG_RECOMMEND_FRIENDS | `custom_recommend_friends(int,int)` |
| 144 | CLIENT_CUSTOMMSG_OPEN_COL_AWARD | `custom_clone_request_open_col_award()` |
| 145 | CLIENT_CUSTOMMSG_PICKUP_CLONEITEM | `custom_pickup_cloneitem(int)` |
| 149 | CLIENT_CUSTOMMSG_BOSS_STRENGTH_CHANGE | `custom_clone_reduce_difficulty_choice(var,var)` |
| 150 | CLIENT_CUSTOMMSG_JINGMAI_WUJI | `custom_jingmai_wuji_msg(int,...)` |
| 151 | CLIENT_CUSTOMMSG_HOME | `custom_home(...)`; `send_home_depot_msg(...)`; `custom_msg_home_gouhuo(...)` |
| 152 | CLIENT_CUSTOMMSG_REQUEST_ADVE_SUMMON | `custom_jianghu_help(wstr)` |
| 155 | CLIENT_CUSTOMMSG_CHITCHAT_GET_PLAYER_INFO | `custom_get_chater_info(wstr,int)` |
| 157 | CLIENT_CUSTOMMSG_ACTIVITY_POINT | `custommsg_activity_point(int,...)` |
| 162 | CLIENT_CUSTOMMSG_HUAREN_CHARGE | `custom_open_huaren_charge(int)` |
| 164 | CLIENT_CUSTOMMSG_CLONESTORE_PURCHASE | `custom_clonestore_purchase(int,str)` |
| 165 | CLIENT_CUSTOMMSG_OPEN_MOUNT_SHOP | `custom_open_mount_shop(int)` |
| 166 | CLIENT_CUSTOMMSG_BAG_ASSISTANT | `custom_bag_assistant(int,...)` |
| 167 | CLIENT_CUSTOMMSG_FIGHT_STATISTICS | `custom_send_fight_statistics(...)` |
| 169 | CLIENT_CUSTOMMSG_NEWSCHOOL_MEET | `custom_send_newschool_shimen(int,...)` |
| 172 | CLIENT_CUSTOMMSG_PREPARE_JINGMAI | `custom_prepare_jingmai_active(int,...)` |
| 173 | CLIENT_CUSTOMMSG_LINXIAO_EXCHANGE | `custom_linxiao_exchange(...)` |
| 180 | CLIENT_CUSTOMMSG_CARD | `custom_card(int,...)` |
| 182 | CLIENT_CUSTOMMSG_ACTIVITY_MANAGE | `custom_get_power_num(int)`; `custom_get_power_num_ex(int)`; `custom_power_redeem(int,var)`; `custom_get_novice_num(int)` +8 |
| 183 | CLIENT_CUSTOMMSG_TEACHER_PUPIL | `custom_teacher_pupil(int,...)` |
| 184 | CLIENT_CUSTOMMSG_DBOMALL_MESSAGE | `custom_dbomall_request(int,...)` |
| 185 | CLIENT_CUSTOMMSG_NEW_TEACHER | `custom_teacher_pupil_new(int,...)` |
| 186 | CLIENT_CUSTOMMSG_ACTIVITY_AWARD | `custom_get_activity_value_rank_prize_rec(int)` |
| 187 | CLIENT_CUSTOMMSG_ATTRIBUTE_MALL | `custom_attr_mall(int,...)` |
| 190 | CLIENT_CUSTOMMSG_GET_CHALLENGE_RANK_BY_PAGE | `custom_update_chellenge_rank_by_page(int)` |
| 191 | CLIENT_CUSTOMMSG_GET_CHALLENGE_RANK_BY_NAME | `custom_update_chellenge_rank_by_name(wstr)` |
| 192 | CLIENT_CUSTOMMSG_GET_CHALLENGE_TARGET_LIST | `custom_update_chellenge_target_list()` |
| 193 | CLIENT_CUSTOMMSG_GET_PLAYER_POSITION | `custom_get_player_position(wstr)` |
| 194 | CLIENT_CUSTOMMSG_CHALLENGE_SET_NICKNAME | `custom_chellenge_set_nickname(var)` |
| 197 | CLIENT_CUSTOMMSG_NPC_RESTART_AI | `custom_request_restart_ai(obj)` |
| 198 | CLIENT_CUSTOMMSG_WORLD_RANK | `send_query_cur_rank(int,str,i64)`; `custom_send_world_rank_query_no(int,str,int)` |
| 199 | CLIENT_CUSTOMMSG_NEIGONGBIND | `custom_update_ng_bind(var)` |
| 200 | CLIENT_CUSTOMMSG_SETSHORTCUT | `custom_set_shortcut(int,var,var)` |
| 201 | CLIENT_CUSTOMMSG_REMOVESHORTCUT | `custom_remove_shortcut(int)` |
| 202 | CLIENT_CUSTOMMSG_OPEN_DIRECTDEPOT | `custom_open_depot()` |
| 205 | CLIENT_CUSTOMMSG_SAVECAPITAL | `custom_save_money_to_bank(int,int)` |
| 206 | CLIENT_CUSTOMMSG_UNSAVECAPITAL | `custom_get_money_from_bank(int,int)` |
| 208 | CLIENT_CUSTOMMSG_WANYINGJINGTIAN_PRICE | `custom_wanyingjingtian_price(int,...)` |
| 213 | CLIENT_CUSTOMMSG_COMMON_WUXUE_CANWU | `custom_common_wuxue_canwu(int)`; `custom_common_wuxue_canwu(int,...)` |
| 214 | CLIENT_CUSTOMMSG_WUXUE_COLLECT | `custom_wuxue_collect(...)` |
| 222 | CLIENT_CUSTOMMSG_WEAR_JINFA | `custom_wear_jinfa(int,str,str)` |
| 224 | CLIENT_CUSTOMMSG_VER_201904 | `custom_ver_201904(int,...)` |
| 231 | CLIENT_CUSTOMMSG_REQ_CHIP_IN | `custom_neigong_pk_chipin()` |
| 232 | CLIENT_COUSOMMSG_REQUEST_NEIG_PK | `custom_request_neigong_pk(var)` |
| 233 | CLIENT_CUSTOMMSG_JOIN_NEIG_PK | `custom_join__neigong_pk(var)` |
| 235 | CLIENT_CUSTOMMSG_LIVEPOINT_INFO | `custom_hold_faculty(int,int)`; `custom_buy_faculty(int,int,int)` |
| 236 | CLIENT_CUSTOMMSG_ROYAL_TREASURE | `custom_rtm_invite_help(int,wstr)`; `custom_quit_royal_treasure(int)` |
| 241 | CLIENT_CUSTOMMSG_DANCING | `custom_dance()` |
| 243 | CLIENT_CUSTOMMSG_RIDE | `custom_ride(int)` |
| 244 | CLIENT_CUSTOMMSG_RIDE_SPURT | `custom_ride_spurt(int)` |
| 245 | CLIENT_CUSTOMMSG_CHOSE_MOUNT | `custom_select_mount(int)` |
| 246 | CLIENT_CUSTOMMSG_EXPEND_MOUNTSTORE | `custom_expendmountstore()` |
| 250 | CLIENT_CUSTOMMSG_PICK_CROP | `custom_pick_crop(int)` |
| 251 | CLIENT_CUSTOMMSG_PICK_ALL_CROP | `custom_pick_all_crop()` |
| 252 | CLIENT_CUSTOMMSG_PICK_CROP_CLOSE | `custom_pick_crop_close()` |
| 253 | CLIENT_CUSTOMMSG_FORGET_SKILL | `custom_send_forget_skill_msg(int,...)` |
| 254 | CLIENT_CUSTOMMSG_SCENE_JHPK | `custom_send_jhpk_get_rank_info(int,wstr)`; `custom_send_scene_jhpk(int,...)` |
| 255 | CLIENT_CUSTOMMSG_WORLD_BOSS | `custom_send_world_boss(int,...)`; `custom_foeman_infall(int,...)` |
| 256 | CLIENT_CUSTOMMSG_DEL_TAOLU | `custom_send_del_taolu(str)` |
| 262 | CLIENT_CUSTOMMSG_SET_ZHENYAN | `custom_set_zhenyan(obj)` |
| 263 | CLIENT_CUSTOMMSG_HOME_FARM | `custom_send_home_farm(int,...)` |
| 264 | CLIENT_CUSTOMMSG_HOME_MINE | `custom_send_home_mine(int,...)` |
| 265 | CLIENT_CUSTOMMSG_CROSS_RECRUIT | `custom_cross_recruit(int,...)` |
| 266 | CLIENT_CUSTOMMSG_NEW_RIDE | `custom_new_ride(int,...)` |
| 267 | CLIENT_CUSTOMMSG_MOBILE_ID | `custom_mobile_id(int,...)` |
| 268 | CLIENT_CUSTOMMSG_ZLCL_PLUNDER | `custom_zlcl_plunder(int,...)` |
| 271 | CLIENT_CUSTOMMSG_CLOSE_ITEMSKILL | `custom_uninstall_skillitem()` |
| 272 | CLIENT_CUSTOMMSG_AUTO_EQUIP_SHOTWEAPON | `custom_auto_equip_shotweapon(var)` |
| 274 | CLIENT_CUSTOMMSG_TRAINPAT_CELLPAT | `custom_zhongqiu_subgroup(int,int)` |
| 280 | CLIENT_CUSTOMMSG_BRIBE | `custom_send_bribe(int,int)` |
| 281 | CLIENT_CUSTOMMSG_ACTIVITY_COLLECT_PIECES | `custom_activity_collect_pieces(int,...)` |
| 289 | CLIENT_CUSTOMMSG_RELIVE_CHECK | `custom_relive_check()` |
| 290 | CLIENT_CUSTOMMSG_RELIVE | `custom_relive(int,var,var)` |
| 291 | CLIENT_CUSTOMMSG_SETPKMODE | `custom_set_pkmode(var)` |
| 292 | CLIENT_CUSTOMMSG_CHANGEWARNING | `custom_change_warning()` |
| 293 | CLIENT_CUSTOMMSG_SETPKPROTECT | `custom_set_pkprotect(var,var)` |
| 295 | CLIENT_CUSTOMMSG_SETTITLE | `custom_set_title(var)` |
| 298 | CLIENT_CUSTOMMSG_REFRESH_TASK_NPC_EFFECT | `custom_refresh_npc_effect(int,obj)` |
| 300 | CLIENT_CUSTOMMSG_LOOK_PLAYER | `custom_on_player_ready(obj)` |
| 302 | CLIENT_CUSTOMMSG_INTERACTIVE_MSG | `custom_send_interact_msg(int,...)`; `custom_send_to_spy(int,wstr)` |
| 303 | CLIENT_CUSTOMMSG_NEW_NEIGPK | `custom_send_new_neigong_pk_msg(int,...)` |
| 304 | CLIENT_CUSTOMMSG_GMCC_MSG | `custom_send_gmcc_msg(int,...)` |
| 305 | CLIENT_CUSTOMMSG_CHECKED_SELECTROLE | `custom_checked_selectrole(int)` |
| 306 | CLIENT_CUSTOMMSG_UNENTHRALL | `request_unenthrall(...)` |
| 307 | CLIENT_CUSTOMMSG_DEPOT_MSG | `send_depot_msg(...)` |
| 308 | CLIENT_CUSTOMMSG_TEAM_MEMBER_SET | `custom_team_member_set(int,int,wstr)` |
| 309 | CLIENT_CUSTOMMSG_TEAM_TYPE_SET | `custom_change_team_type(int)` |
| 310 | CLIENT_CUSTOMMSG_TEAM_CREATE | `custom_team_create()` |
| 311 | CLIENT_CUSTOMMSG_TEAM_KICKOUT | `custom_kickout_team(wstr)` |
| 312 | CLIENT_CUSTOMMSG_TEAM_LEAVE | `custom_leave_team()` |
| 313 | CLIENT_CUSTOMMSG_TEAM_CHANGE_CAPTAIN | `custom_caption_change(wstr)` |
| 314 | CLIENT_CUSTOMMSG_TEAM_SET_INFO/CLIENT_CUSTOMMSG_TEAM_SET_PICK_QUALITY | `custom_set_team_allot_quality(int)` |
| 315 | CLIENT_CUSTOMMSG_TEAM_SET_PICK_MODE | `custom_set_team_allot_mode(int)` |
| 316 | CLIENT_CUSTOMMSG_TEAM_REQUEST_JOIN | `custom_team_request_join(wstr)` |
| 317 | CLIENT_CUSTOMMSG_TEAM_REFRESH_INFO | `custom_team_info_refresh()` |
| 318 | CLIENT_CUSTOMMSG_TEAM_ADD_INFO | `custom_team_add_info(wstr,wstr,wstr)` |
| 319 | CLIENT_CUSTOMMSG_TEAM_REMOVE_INFO | `custom_team_remove_info()` |
| 320 | CLIENT_CUSTOMMSG_EXCHANGE_OK | `custom_exchange_ok()` |
| 321 | CLIENT_CUSTOMMSG_EXCHANGE_LOCK | `custom_exchange_lock()` |
| 322 | CLIENT_CUSTOMMSG_EXCHANGE_CANCEL | `custom_exchange_cancel(int)` |
| 323 | CLIENT_CUSTOMMSG_EXCHANGE_MONEY | `custom_exchange_setmoney(var,var)` |
| 324 | CLIENT_CUSTOMMSG_EXCHANGE_ADD_ITEM | `custom_exchange_add_item(int,int,int,int)` |
| 325 | CLIENT_CUSTOMMSG_EXCHANGE_REMOVE_ITEM | `custom_exchange_remove_item(int)` |
| 326 | CLIENT_CUSTOMMSG_TEAM_SET_SIGN | `custom_team_sign_set(str)` |
| 327 | CLIENT_CUSTOMMSG_TASK_MSG | `custom_giveup_task(int,int)`; `send_task_msg(int,...)`; `custom_accept_task(int,int)`; `custom_complete_task(str,int,int,obj)` +7 |
| 350 | CLIENT_CUSTOMMSG_POST_OPENBOX | `custom_open_mail_box(int)` |
| 351 | CLIENT_CUSTOMMSG_POST_CANCEL | `custom_cancel_send_letter()` |
| 352 | CLIENT_CUSTOMMSG_POST_SEND | `custom_send_letter(wstr,wstr,wstr,str,str,str)` |
| 353 | CLIENT_CUSTOMMSG_POST_READ | `custom_read_letter(str)` |
| 357 | CLIENT_CUSTOMMSG_POST_SELECT | `custom_select_letter(var,var,var)` |
| 358 | CLIENT_CUSTOMMSG_POST_CLOSE_READBOX | `custom_close_mail_readbox()` |
| 359 | CLIENT_CUSTOMMSG_POST_GET_APPEDIX | `custom_get_appendix(str)` |
| 362 | CLIENT_CUSTOMMSG_CHITCHAT_WHISPER_READ | `custom_remove_chat_whisper(wstr)` |
| 363 | CLIENT_CUSTOMMSG_POST_SELECT_SEND | `custom_select_send_letter(int,int)` |
| 365 | CLIENT_CUSTOMMSG_POST | `custom_post(int,...)` |
| 366 | CLIENT_CUSTOMMSG_POST_ACCEPT_FRIEND | `custom_send_accept_friend_letter_flag(...)` |
| 367 | CLIENT_CUSTOMMSG_SSG_YJP | `custom_ssg_yjp(int,...)` |
| 368 | CLIENT_CUSTOMMSG_MYSTERIOUS_REALM | `custom_myr(int,...)` |
| 370 | CLIENT_CUSTOMMSG_SINGLEFREETEACHPRNRELATION | `custom_singlefreeteachpren(int,wstr)` |
| 373 | CLIENT_CUSTOMMSG_QUERYTEACHPRNELIST | `custom_queryteachprenlist(int,int)` |
| 374 | CLIENT_CUSTOMMSG_GETLISTBYCONDITION | `custom_send_teacher_commend(int)` |
| 375 | CLIENT_CUSTOMMSG_GETPRENTICE_INCOME | `custom_get_prentice_income(wstr)` |
| 377 | CLIENT_CUSTOMMSG_JUDGETEACH_RESULT | `custom_judge_teach_result(var)` |
| 380 | CLIENT_CUSTOMMSG_PICKUP_DROPITEM | `custom_pickup_single_item(int)` |
| 382 | CLIENT_CUSTOMMSG_CLOSE_DROPBOX | `custom_close_drop_box()` |
| 383 | CLIENT_CUSTOMMSG_FINISH_BASEPRICE | `custom_set_compete_baseprice(obj,int,int)` |
| 384 | CLIENT_CUSTOMMSG_COMPETE_ITEM_RESULT | `custom_compete_item_result(obj,int)` |
| 385 | CLIENT_CUSTOMMSG_SCENE_COMPETE | `custom_send_scene_compete_msg(int,...)` |
| 387 | CLIENT_CUSTOMMSG_ROLL_ITEM_RESULT | `custom_roll_item_result(obj,int)` |
| 388 | CLIENT_CUSTOMMSG_TEAMLEADER_ALLOC_ITEM | `custom_teamleader_alloc_item(int,wstr,int)` |
| 389 | CLIENT_CUSTOMMSG_CHOICEITEM | `custom_choice(int,...)` |
| 390 | CLIENT_CUSTOMMSG_MARRY | `custom_marry(int,...)` |
| 391 | CLIENT_CUSTOMMSG_COMMON_COMPETE | `custom_common_compete(int,...)` |
| 392 | CLIENT_CUSTOMMSG_FANGHUAJIE | `custom_fhj(int,...)` |
| 393 | CLIENT_CUSTOMMSG_INNER_EQUIP | `custom_ine_msg(int,...)` |
| 397 | CLIENT_CUSTOMMSG_SCHOOL_MSG | `custom_select_school(int,...)`; `custom_betray_new_school(int)` |
| 398 | CLIENT_CUSTOMMSG_CLONE_UNIVERSAL_HOARD | `custom_clone_universal_hoard(int,...)` |
| 401 | CLIENT_CUSTOMMSG_REQUEST_START_ESCORT/CLIENT_CUSTOMMSG_REQUEST_START_ESCORT | `custom_request_start_escort(str,int,int,int)` |
| 402 | CLIENT_CUSTOMMSG_ESCORT_MOVE_CONTROL | `custom_request_escort_control(var,var)` |
| 405 | CLIENT_CUSTOMMSG_GET_ESCORT_INFO | `custom_get_escort_info(var)` |
| 407 | CLIENT_CUSTOMMSG_ESCORT_MSG | `custom_escort_anshao_msg(var,var)`; `custom_send_escort_msg(var,...)` |
| 408 | CLIENT_CUSTOMMSG_ISSPECIALTIME | `custom_get_is_special_time(int)` |
| 409 | CLIENT_CUSTOMMSG_JJGG | `custom_jjgg_msg(int,...)` |
| 410 | CLIENT_CUSTOMMSG_START_TRUSTEE | `custom_start_trustee(int)` |
| 411 | CLIENT_CUSTOMMSG_STOP_TRUSTEE | `custom_stop_trustee()` |
| 412 | CLIENT_CUSTOMMSG_SET_TRUSTEE | `custom_set_trustee(int,int)` |
| 413 | CLIENT_CUSTOMMSG_CLEAR_TRUSTEE_LOG | `custom_clear_trustee_log()` |
| 414 | CLIENT_CUSTOMMSG_FLEE_IN_NIGHT | `custom_flee_in_night(int,...)` |
| 415 | CLIENT_CUSTOMMSG_KOF | `custom_kof(int,...)` |
| 416 | CLIENT_CUSTOMMSG_HIDE_PUZZLES | `custom_request_hide_puzzles_faculty(int)`; `custom_request_abort_hide_puzzles_faculty(int)`; `custom_request_hide_puzzles_war(int,...)` |
| 417 | CLIENT_CUSTOMMSG_GUILD_ENTRENCE_WAR | `custom_gew(int,...)` |
| 418 | CLIENT_CUSTOMMSG_SEEK_EMPEROR_CITY | `custom_sec(int,...)` |
| 419 | CLINET_CUSTOMMSG_GOTO | `custom_goto_end(...)` |
| 422 | CLIENT_CUSTOMMSG_CHAMPION_WAR | `custom_champion_war(int,...)`; `custom_request_guildbl(int)`; `custom_request_guildplnum(int)` |
| 423 | CLIENT_CUSTOMMSG_WORLD_AUCTION | `custom_world_auction(int,...)` |
| 424 | CLIENT_CUSTOMMSG_AGREE_WAR | `custom_agree_war(int,...)` |
| 425 | CLIENT_CUSTOMMSG_OUTLAND_WAR | `custom_outland_war(int,...)` |
| 426 | CLIENT_CUSTOMMSG_LEAGUE_MATCHES | `custom_league_matches(int,...)` |
| 427 | CLIENT_CUSTOMMSG_BODYGUARDOFFICE | `custom_send_bodyguardoffice_sender(int,...)` |
| 428 | CLIENT_CUSTOMMSG_GUILD_PEAK_WAR | `custom_gpw(int,...)` |
| 429 | CLENIT_CUSTOMMSG_QUERY_INV_RANK | `custom_query_inv_rank(int,str)`; `custom_query_friend_rank(int,str,str,wstr)` |
| 430 | CLIENT_CUSTOMMSG_REQUEST_ACTION | `custom_request_action(str)` |
| 432 | CLIENT_CUSTOMMSG_MUTUAL_ACTION | `custom_over_mutual_action(int)` |
| 433 | CLIENT_CUSTOMMSG_TAROT | `custom_tarot(int,...)` |
| 437 | CLIENT_CUSTOMMSG_ASSEMBLE_STONE | `custom_assemble_stone(int,...)` |
| 438 | CLIENT_CUSTOMMSG_SHADOW_MIRROR | `custom_shm(int,...)` |
| 439 | CLIENT_CUSTOMMSG_WAR_ORDER | `custom_war_order(int,...)` |
| 441 | CLIENT_CUSTOMMSG_SCHOOL_JAP | `custom_sjap(int,...)` |
| 442 | CLIENT_CUSTOMMSG_TEAM_PLAYER_OPEN | `custom_team_player_open()` |
| 443 | CLIENT_CUSTOMMSG_TEAM_PLAYER_CLOSE | `custom_team_player_close()` |
| 444 | CLIENT_CUSTOMMSG_TEAM_DESTROY | `custom_team_destroy()` |
| 445 | CLIENT_CUSTOMMSG_TEAM_INVITE | `custom_team_invite(wstr)` |
| 446 | CLIENT_CUSTOMMSG_OVER_CLOUD | `custom_over_cloud(int,...)` |
| 447 | CLIENT_CUSTOMMSG_CHALLENGE_CLONE | `custom_challenge_clone(int,...)` |
| 449 | CLIENT_CUSTOMMSG_MOJUE | `custom_mojue(int,...)` |
| 450 | CLIENT_CUSTOMMSG_GHW | `custom_ghw(int,...)` |
| 451 | CLIENT_CUSTOMMSG_BUY_CAPITAL | `custom_buy_capital(...)` |
| 452 | CLIENT_CUSTOMMSG_GWD | `custom_gwd(int,...)` |
| 453 | CLIENT_CUSTOMMSG_DONGFANG | `custom_dongfang_msg(int,...)` |
| 454 | CLIENT_CUSTOMMSG_PUNISH | `custom_msg_pk_punish(int,str)` |
| 455 | CLIENT_CUSTOMMSG_SCHOOL_DANCE | `custom_send_to_shool_dance(int)` |
| 457 | CLIENT_CUSTOMMSG_ANCIENT_TOMB | `custom_ancient_tomb_sender(int,...)` |
| 458 | CLIENT_CUSTOMMSG_HUASHAN_SCHOOL | `custom_huashan_school(int,...)` |
| 459 | CLIENT_CUSTOMMSG_TGROLL_GAME | `custom_tg_roll_game(int,...)` |
| 460 | CLIENT_CUSTOMMSG_IMPART_BEGIN | `custom_begin_impart(var)` |
| 461 | CLIENT_CUSTOMMSG_IMPART_CANCEL | `custom_cancel_impart()` |
| 463 | CLIENT_CUSTOMMSG_SEA_LOST_TEMPLE | `custom_sea_lost_temple(int,...)` |
| 464 | CLIENT_CUSTOMMSG_CROSS_ESCORT | `custom_cem(int,...)` |
| 477 | CLIENT_CUSTOMMSG_SWORN | `custom_sworn(int,...)` |
| 479 | CLIENT_CUSTOMMSG_STAR | `custom_query_star_info(int,int,str)` |
| 495 | CLIENT_CUSTOMMSG_CANOPEN_FORTUNETELLING | `custom_canopen_fortunetelling()` |
| 499 | CLIENT_CUSTOMMSG_WQGAME_LOOK | `custom_wqgame_look(var)` |
| 502 | CLIENT_CUSTOMMSG_CLFROLLGAME | `custom_send_clf_roll_game(...)` |
| 503 | CLIENT_CUSTOMMSG_HUAWUQUE_RACE | `custom_send_hwq_choice(...)` |
| 504 | CLIENT_CUSTOMMSG_IT_TAKES_TWO | `custom_itt_msg(int,...)` |
| 508 | CLIENT_CUSTOMMSG_NORMAL_PET | `custom_normal_pet(...)` |
| 509 | CLIENT_CUSTOMMSG_SSG_SCHOOLMEET | `custom_ssg_schoolmeet(int,...)` |
| 511 | CLIENT_CUSTOMMSG_WXJ_SCHOOLMEET | `custom_wxj_schoolmeet(int,...)` |
| 512 | CLIENT_CUSTOMMSG_DMP_SCHOOLMEET | `custom_dmp_schoolmeet(int,...)` |
| 514 | CLIENT_CUSTOMMSG_GUIXU_BOSS | `custom_gxb(int,...)` |
| 517 | CLIENT_CUSTOMMSG_GET_NPC_KARMA_VALUE | `custom_get_npc_karma_value(str)`; `custom_get_npc_karma_value(str)` |
| 518 | CLIENT_CUSTOMMSG_MAIN_KARMA | `apply_add_npc_relation(int,str,int,str)`; `apply_add_npc_relation(int,str,int)` |
| 519 | CLIENT_CUSTOMMSG_DELETE_PLOT | `custom_delete_plot(int)` |
| 520 | CLIENT_CUSTOMMSG_GET_NPC_KARMA | `custom_query_npc_karma(str,int,int)` |
| 521 | CLIENT_CUSTOMMSG_REQUEST_LEAVE_YHG_TD | `custom_yhg_td_request_leave()` |
| 522 | CLIENT_CUSTOMMSG_ZLCL | `custom_zlcl_msg(int,...)` |
| 529 | CLIENT_CUSTOMMSG_SHOW_NPC_FRIEND | `custom_show_npc_friend()` |
| 530 | CLIENT_CUSTOMMSG_AUTO_PRIZE_BOX | `custom_auto_show_prize_box(int)` |
| 531 | CLIENT_CUSTOMMSG_PUBLISH_COLLECT_TASK | `custom_publish_collect_task(wstr,str,int,int,int)` |
| 532 | CLIENT_CUSTOMMSG_OPEN_PUBLISH_FORM | `custom_open_publish_form()` |
| 533 | CLIENT_CUSTOMMSG_OPEN_QUERY_EDIT_FORM | `custom_open_query_edit_task()` |
| 534 | CLIENT_CUSTOMMSG_QUERY_TYPE_TASK | `custom_query_type_task(int)` |
| 535 | CLIENT_CUSTOMMSG_QUERY_PREV_NEXT_PAGE | `custom_task_prev_next_page(str,int,int)` |
| 536 | CLIENT_CUSTOMMSG_QUERY_SELECT_TASK_INFO | `custom_query_select_edittask_info(int)` |
| 537 | CLIENT_CUSTOMMSG_ACCEPT_EDIT_TASK | `custom_accept_edit_task(int)` |
| 538 | CLIENT_CUSTOMMSG_OPEN_EDIT_FORM_ACCEPTED | `custom_open_edit_task_accepted()` |
| 539 | CLIENT_CUSTOMMSG_GIVEUP_EDIT_TASK | `custom_giveup_edittask(int,wstr)` |
| 540 | CLIENT_CUSTOMMSG_SUBMIT_EDIT_TASK | `custom_submit_edittask(int)` |
| 541 | CLIENT_CUSTOMMSG_SHOW_SUBMIT_BOX | `custom_show_submit_collect_box(int)` |
| 542 | CLIENT_CUSTOMMSG_SUBMIT_COLLECT_EDITTASK | `custom_submit_collect_edit_task(int)` |
| 550 | CLIENT_CUSTOMMSG_CREATE_WORLD_TRANS_TOOL | `custom_create_world_trans_tool(int,str)` |
| 552 | CLIENT_CUSTOMMSG_NEW_SCHOOL | `custom_school(int,...)` |
| 553 | CLIENT_CUSTOMMSG_CHALLENGE_HILL | `custom_challenge_hill(var,...)` |
| 554 | CLIENT_CUSTOMMSG_REQUEST_WUJIDAO | `custom_wjd_request(int)` |
| 555 | CLIENT_CUSTOMMSG_SNS_ENEMY_INFO_RESULT | `custom_query_enemy_info(int,int,wstr)` |
| 556 | CLIENT_CUSTOMMSG_ACCOUNT_CHANGE | `custom_send_name_change(wstr,int)` |
| 558 | CLIENT_CUSTOMMSG_BALANCE_PK | `custom_balance_pk(int,...)` |
| 559 | CLIENT_CUSTOMMSG_PAINTING_DEPOT | `custom_painting_depot(int,...)` |
| 561 | CLIENT_CUSTOMMSG_EXTRA_SKILL | `custom_extra_skill(...)` |
| 562 | CLIENT_CUSTOMMSG_COPY_SKILL | `custom_copy_skill(...)` |
| 565 | CLIENT_CUSTOMMSG_DRAW_PRIZE | `custom_draw_prize(int)` |
| 566 | CLIENT_CUSTOMMSG_FIVE_SKY | `custom_five_sky(int,...)` |
| 567 | CLIENT_CUSTOMMSG_GHOST_CITY | `custom_ghost_city(int,...)` |
| 568 | CLIENT_CUSTOMMSG_DEFEND_TEMPLE | `custom_defend_temple(int,...)` |
| 569 | CLIENT_CUSTOMMSG_CROSS_INVADE | `custom_cross_invade(int,...)` |
| 571 | CLIENT_CUSTOMMSG_ISLAND_PLUNDER_WAR | `custom_island_plunder_war(int,...)` |
| 572 | CLIENT_CUSTOMMSG_HOME_WORKSHOP | `custom_home_workshop_msg(int,...)` |
| 573 | CLIENT_CUSTOMMSG_ACTIVE_DMFS | `custom_act_dmfs_msg(int,...)` |
| 574 | CLIENT_CUSTOMMSG_SANDSEA_CONVOY | `custom_sandsea_convoy(int,...)` |
| 575 | CLIENT_CUSTOMMSG_SCENE_SCHOOL_TASK | `custom_scene_school_task(int,...)` |
| 600 | CLIENT_CUSTOMMSG_COMPOSITE | `custom_send_compose(var,var,var,var,var)` |
| 601 | CLIENT_CUSTOMMSG_BEG | `custom_send_beg(str)` |
| 602 | CLIENT_CUSTOMMSG_SUANGUA | `custom_send_suangua()` |
| 603 | CLIENT_CUSTOMMSG_OPEN_COMPOSITEBOX | `custom_send_open_compositebox(str)` |
| 604 | CLIENT_CUSTOMMSG_CLOSE_COMPOSITEBOX | `custom_send_close_compositebox()` |
| 605 | CLIENT_CUSTOMMSG_COMPOSITEBOX_ADDITEM | `custom_send_input_compositebox(int,int)` |
| 606 | CLIENT_CUSTOMMSG_COMPOSITEBOX_REMOVEITEM | `custom_send_del_item_compositebox(int)` |
| 607 | CLIENT_CUSTOMMSG_COMPOSITEBOX_AUTO_PUT | `custom_send_composite_auto_put(str)` |
| 612 | CLIENT_CUSTOMMSG_LIFE_SHARE_COMPOSITE_ADD | `custom_add_life_share(str,str,str)` |
| 613 | CLIENT_CUSTOMMSG_LIFE_SHARE_COMPOSITE_DEL | `custom_del_life_share(str,int)` |
| 614 | CLIENT_CUSTOMMSG_LIFE_SHARE_COMPOSITE_MODIFY | `custom_set_life_compose_share(str,str)` |
| 615 | CLIENT_CUSTOMMSG_LIFE_SHARE_COMPOSITE_CLEAR | `custom_clear_life_share(str)` |
| 616 | CLIENT_CUSTOMMSG_LIFE_SHARE_REQUEST_LOOK | `custom_life_request_look(wstr,str)` |
| 617 | CLIENT_CUSTOMMSG_LIFE_SHARE_LIFE_SHARE_STOP | `custom_stop_life_trade()` |
| 618 | CLIENT_CUSTOMMSG_FUSE | `custom_send_fuse(var,var,var,var)` |
| 619 | CLIENT_CUSTOMMSG_LIFE_SHARE_SKILL_MODIFY | `custom_set_life_skill_share(str,str)` |
| 620 | CLIENT_CUSTOMMSG_CANCEL_FUSE | `custom_cancel_fuse()` |
| 621 | CLIENT_CUSTOMMSG_BEGPLAYER | `custom_send_begplayer(obj,int)` |
| 622 | CLIENT_CUSTOMMSG_LIFE_SHARE_REQUEST | `custom_request_life_share(int,var,...)` |
| 623 | CLIENT_CUSTOMMSG_LIFE_SHARE_LIFE_SHARE_START | `custom_start_life_trade()` |
| 624 | CLIENT_CUSTOMMSG_LIFE_QIS_EQUIP_ZHANSHU | `custom_life_qis_equip_zhanshu(str)` |
| 625 | CLIENT_CUSTOMMSG_LIFE_QIS_CLEAR_EQUIP_ZHANSHU | `custom_life_qis_clear_equip_zhanshu()` |
| 626 | CLIENT_CUSTOMMSG_ADD_SPLITITEM | `custom_add_split_item(int,int)` |
| 627 | CLIENT_CUSTOMMSG_REMOVE_SPLITITEM | `custom_remove_split_item()` |
| 629 | CLIENT_CUSTOMMSG_LIFESKILL_SPLITITEM | `custom_lifeskill_split_item(str)` |
| 630 | CLIENT_CUSTOMMSG_SPLIT_BOOK | `custom_start_split_book(int)`; `custom_cancel_split_book(int)`; `custom_add_split_book(int,var,var,var)`; `custom_remove_split_book(int,var)` |
| 631 | CLIENT_CUSTOMMSG_CONTRIBUTE | `custom_start_contribute(int)`; `custom_cancel_contribute(int)`; `custom_add_contribute_item(int,var,var,var)`; `custom_remove_contribute_item(int,var)` |
| 633 | CLIENT_CUSTOMMSG_NPCEXCHANGE | `custom_start_npc_exchange(int,var)`; `custom_cancel_npc_exchange(int)`; `custom_add_npc_exchange_item(int,var,var,var)`; `custom_remove_npc_exchange_item(int,var)` |
| 634 | CLIENT_CUSTOMMSG_SKILLPAGE_FUSE | `custom_send_skillpage_fuse(var,var,var)` |
| 636 | CLIENT_CUSTOMMSG_INPHASE_CLONE | `custom_inphase_clone(str)` |
| 637 | CLIENT_CUSTOMMSG_UPGRADE_EQUIP | `custom_upgrade_equip(int,...)` |
| 638 | CLIENT_CUSTOMMSG_TREASUREBOX_MSG | `custom_treasurebox_msg(...)` |
| 639 | CLIENT_CUSTOMMSG_CONVERT_EQUIP | `custom_chang_equip_wx(int,...)` |
| 640 | CLIENT_CUSTOMMSG_QIMEN_EQUIP | `custom_chang_equip_qm(int,...)` |
| 656 | CLIENT_CUSTOMMSG_OFFLINE_CLOSE_STALL | `custom_stall_return_ready()` |
| 658 | CLIENT_CUSTOMMSG_OFFLINE_BUY_OTHERONE_STALL | `custom_stall_buy_from_stall(wstr,str,int,i64)` |
| 685 | CLIENT_CUSTOMMSG_BEGIN_COLLECT_EQUIP | `custom_equipblend_collect(int,int)` |
| 687 | CLIENT_CUSTOMMSG_COLLECT_EQUIP | `custom_collect_equip(int,...)` |
| 688 | CLIENT_CUSTOMMSG_TERSURE | `custom_tersure_lock(int,int,int)`; `custom_tersure_unlock(int,int,int)`; `custom_tersure_target_lock(int,int,int)`; `custom_tersure_target_unlock(int,int,int)` +25 |
| 689 | CLIENT_CUSTOMMSG_ATTIRE_EQUIP_COPY | `custom_attire_equip_copy_msg(str,int,int)` |
| 692 | CLIENT_CUSTOMMSG_ATTIRE_EQUIP_BLEND | `custom_attire_equip_blend(int,str)` |
| 693 | CLIENT_CUSTOMMSG_LINGXIAO_ZHIZUN | `custom_linxiao_zhizun(int,...)` |
| 694 | CLIENT_CUSTOMMSG_CHECK_DELETE_FORCE | `custom_check_delete_force(int,...)` |
| 701 | CLIENT_CUSTOMMSG_SET_SHOW_ORIGIN | `custom_send_show_origin_type(str,int)` |
| 702 | CLIENT_CUSTOMMSG_GET_ORIGIN_PRIZE | `custom_send_get_origin_prize(var)`; `custom_form_outland_send_get_item(int)` |
| 705 | CLIENT_CUSTOMMSG_GET_ORIGIN | `custom_send_get_origin(str)`; `custom_form_outland_send_get_origin(int)` |
| 707 | CLIENT_CUSTOMMSG_SET_FACE | `custom_face(...)` |
| 711 | CLIENT_CUSTOMMSG_REGISTER | `custom_register(...)` |
| 712 | CLIENT_CUSTOMMSG_SWEET_PET_CLOTHES | `custom_msg_sweet_pet_clothes(int,str)` |
| 713 | CLIENT_CUSTOMMSG_BOX_EVENT | `custom_box_event(int,...)` |
| 716 | CLIENT_CUSTOMMSG_OUTLAND_EVENT | `custom_outland_event_select_item(int,...)` |
| 718 | CLIENT_CUSTOMMSG_OUTLAND_VALUE_AWARD | `custom_outland_reward(int,int)` |
| 719 | CLIENT_CUSTOMMSG_SKY_HILL | `custom_skyhill_achievement(int,int)`; `custom_skyhill_get_reward(int,int)`; `custom_skyhill_next(int)`; `custom_form_ttf_template(int)` |
| 720 | CLIENT_CUSTOMMSG_PIKONGZHANG_ACTIVITY | `custom_pikongzhang_activity(int,...)` |
| 723 | CLIENT_CUSTOMMSG_SCHOOL_EXTINCT | `custom_school_counter_attack_data(int)`; `custom_shcool_destroy(int)`; `custom_protect_school_step(int)`; `custom_protect_school_time(int,int)` +2 |
| 724 | CLIENT_CUSTOMMSG_GUILD_LUCK_PRAY | `custom_guild_luck_pray(int,...)` |
| 725 | CLIENT_CUSTOMMSG_TEACH_EXAM | `custom_teach_exam(int,...)` |
| 726 | CLIENT_CUSTOMMSG_SUPER_MARKET | `custom_super_market_msg(int,...)` |
| 727 | CLIENT_CUSTOMMSG_AUCTION_MSG | `custom_auction(int,...)` |
| 754 | CLIENT_CUSTOMMSG_LEITAI_REWARD_ITEM_INFO | `custom_send_leitai_reward_info(int,int,var)` |
| 755 | CLIENT_CUSTOMMSG_LEITAI_REWARD_COUNT | `custom_send_leitai_integral_num()` |
| 756 | CLIENT_CUSTOMMSG_LEITAI_QIECUO_COUNT | `custom_send_leitai_qiecuo_num()` |
| 757 | CLIENT_CUSTOMMSG_LEITAI_INTEGRAL_INFO | `custom_send_leitai_integral_info()` |
| 758 | CLIENT_CUSTOMMSG_LEITAI_WAR | `custom_send_leitai_wager_info(int,wstr,int,var)`; `custom_send_leitai_info(int,...)`; `custom_send_leitai_apply(int)`; `custom_send_leitai_filter_Info_request(int)` |
| 759 | CLIENT_CUSTOMMSG_CHALLENGE_LOSE | `custom_send_challenge_lose()` |
| 760 | CLIENT_CUSTOMMSG_WORD_CHECK | `custom_set_second_word(int,wstr,wstr)`; `modify_second_word(int,wstr)`; `cancel_second_word(int,wstr)`; `set_password_free(int,wstr)` +7 |
| 761 | CLIENT_CUSTOMMSG_CHAT_MSG | `custom_get_cross_speaker_data(int,...)`; `custom_get_speaker_rank(int)`; `custom_save_chatpage_config(int,wstr)`; `custom_request_silence(int,int,wstr)` |
| 762 | CLIENT_CUSTOMMSG_CHANGE_EQUIP_ATTRIBUTE | `custom_start_change_equip(int,var,var)`; `custom_cancel_change_equip(int)`; `custom_add_change_equip(int,var,var,var)`; `custom_remove_change_equip(int,var)` |
| 765 | CLIENT_CUSTOMMSG_REPENT | `end_repent(int)` |
| 766 | CLIENT_CUSTOMMSG_SAN_HILL | `custom_sanhill_msg(int,...)` |
| 767 | CLIENT_CUSTOMMSG_SINGLE_SHOP | `custom_single_shop_msg(int,...)` |
| 768 | CLIENT_CUSTOMMSG_WHZ_SCHOOL_MEET | `custom_send_whz_shimen(int,...)` |
| 769 | CLIENT_CUSTOMMSG_JZSJ_QHYC | `custom_send_jzsj_qhyc(...)` |
| 770 | CLIENT_CUSTOMMSG_JIUYINZHI_GET_INFO | `custom_query_jiuyinzhi_current_step(str,int)`; `custom_send_jyz_msg(var,int,...)` |
| 771 | CLIENT_CUSTOMMSG_ROUND_TASK | `custom_send_query_round_task(int,int)` |
| 773 | CLIENT_CUSTOMMSG_TYHG_SCHOOL_MEET | `custom_send_tyhg_shimen(int,...)` |
| 774 | CLIENT_CUSTOMMSG_RED_PACKET | `custom_red_packet(...)` |
| 775 | CLIENT_CUSTOMMSG_GUILD_STATION_ACTIVITY | `custom_send_guild_station_activity(...)` |
| 776 | CLIENT_CUSTOMMSG_GUILD_CROSS_WAR | `custom_cross_guild_war(int,...)` |
| 777 | CLIENT_CUSTOMMSG_SJY_SCHOOL_MEET | `custom_send_sjy_shimen(int,...)` |
| 778 | CLIENT_CUSTOMMSG_XMG_SCHOOL_MEET | `custom_send_xmg_shimen(int,...)` |
| 783 | CLIENT_CUSTOMMSG_WUDAO_WAR | `custom_wudao(int,...)` |
| 784 | CLIENT_CUSTOMMSG_LUAN_DOU | `custom_luandou(int,...)` |
| 785 | CLIENT_CUSTOMMSG_JIUYANG_FACULTY | `custom_jiuyang_faculty(int,...)`; `custom_quit_jiuyang_faculty(int)` |
| 786 | CLIENT_CUSTOMMSG_TAOSHA | `custom_taosha(int,...)` |
| 787 | CLIENT_CUSTOMMSG_SEE | `custom_see(int,...)` |
| 788 | CLIENT_CUSTOMMSG_GROUP_CHAT | `custom_create_groupchat(int,wstr,...)`; `custom_del_groupchat(int,str)`; `custom_open_groupchat(int,str)`; `custom_groupchat_add_member(int,str,wstr,wstr)` +4 |
| 789 | CLIENT_CUSTOMMSG_WUDAO_YANWU | `custom_wudao_yanwu(int,...)` |
| 790 | CLIENT_CUSTOMMSG_CROSS_STATION_WAR | `custom_corss_station_war(int,...)` |
| 801 | CLIENT_CUSTOMMSG_CITY_DEFENSE | `custom_send_guild_city_def(...)` |
| 803 | CLIENT_CUSTOMMSG_GUILD_GLOBAL_WAR | `custom_guildglobalwar(int,...)` |
| 804 | CLIENT_CUSTOMMSG_ACTIVITY_HORSE_RACE | `custom_horse_race(int,...)` |
| 805 | CLIENT_CUSTOMMSG_BEFORE_END_MOVIE | `custom_before_movie_end(int)` |
| 806 | CLIENT_CUSTOMMSG_END_MOVIE | `custom_movie_end(int)` |
| 807 | CLIENT_CUSTOMMSG_AGREE_MOVIE | `custom_agree_movie()` |
| 808 | CLIENT_CUSTOMMSG_START_MOVIE | `custom_movie_start(obj,int)` |
| 821 | CLIENT_CUSTOMMSG_OFFLINE_SET_FREE | `custom_offline_setfree(int,wstr)` |
| 823 | CLIENT_CUSTOMMSG_OFFLINE_BOOK_KEEPER_STUN | `stun_offline_book_keeper(wstr)` |
| 825 | CLIENT_CUSTOMMSG_CHECK_SWITCH_ENABLE | `custom_switch_msg(int)` |
| 826 | CLIENT_CUSTOMMSG_QUERY_HOLIDAY_ACTIVE | `custom_req_active()` |
| 829 | CLIENT_CUSTOMMSG_EGWAR | `custom_egwar_trans(int,...)` |
| 831 | CLIENT_CUSTOMMSG_SWEET_EMPLOY | `custom_offline_employ(int,...)` |
| 833 | CLIENT_CUSTOMMSG_JH_EXPLORE | `custom_get_tourprize(...)` |
| 835 | CLIENT_CUSTOMMSG_NEW_SCHOOL_FIGHT | `custom_cross_school(...)` |
| 837 | CLIENT_CUSTOMMSG_FESTIVAL_SIGN | `custom_request_del_appoint_item(int,...)`; `custom_request_festival_sign_info(int,...)`; `custom_request_festival_sign_acctask(int,int)`; `custom_request_festival_sign_get_big(int)` +2 |
| 838 | CLIENT_CUSTOMMSG_MIDDLE_SPRING_HUNT | `custom_quit_middle_spring_hunt(int)` |
| 839 | CLIENT_CUSTOMMSG_GROWGIFT | `custom_grow_gift(int,...)` |
| 840 | CLIENT_CUSTOMMSG_ACTIVE_WWZJ | `custom_act_wwzj(int,...)` |
| 841 | CLIENT_CUSTOMMSG_ACTIVE_SLHJ | `custom_act_slhj(int,...)` |
| 842 | CLIENT_CUSTOMMSG_ACTIVE_JGTB | `custom_act_jgtb(int,...)` |
| 847 | CLIENT_CUSTOMMSG_LEITAI_COMPETE | `request_quit_leitaiconpete(int)` |
| 848 | CLIENT_CUSTOMMSG_OFFLINE_ACCOST | `custom_offline_accost(int,...)` |
| 849 | CLIENT_CUSTOMMSG_AVENGE | `custom_avenge(int,...)` |
| 851 | CLIENT_CUSTOMMSG_OP_FISHING | `custom_op_fishing()` |
| 852 | CLIENT_CUSTOMMSG_BEGIN_COPY | `custom_send_chaolu(int,var)` |
| 853 | CLIENT_CUSTOMMSG_PREPARE_FORTUNETELLING | `custom_repare_fortunetelling()` |
| 854 | CLIENT_CUSTOMMSG_BEGIN_FORTUNETELLING | `custom_begin_fortunetelling(wstr,int,wstr,int,int,int)` |
| 855 | CLIENT_CUSTOMMSG_END_FORTUNETELLING | `custom_stop_fortunetelling()` |
| 856 | CLIENT_CUSTOMMSG_REQUEST_FORTUNETELLING | `custom_request_fortunetelling(wstr)` |
| 857 | CLIENT_CUSTOMMSG_BEGIN_ANNOTATE | `custom_send_annotate(int,int,var)`; `custom_annotate_fresh_function(int,int,int,...)` |
| 860 | CLIENT_CUSTOMMSG_BEGIN_QIN | `custom_doqin(str)` |
| 861 | CLIENT_CUSTOMMSG_BEGIN_QIANGHUA | `custom_send_qianghua(int,int,str,str)` |
| 862 | CLIENT_CUSTOMMSG_SPLIT_ITEM | `custom_split_item(int,str)` |
| 863 | CLIENT_CUSTOMMSG_CANCEL_SPLIT_ITEM | `custom_cancel_split_item()` |
| 864 | CLIENT_CUSTOMMSG_JOB | `custom_giveup_job(int,var)`; `custom_study_job(int,...)`; `custom_study_job_upgrade(int,...)` |
| 865 | CLIENT_CUSTOMMSG_TIGUAN | `custom_tiguan(int,...)` |
| 869 | CLIENT_CUSTOMMSG_PREPARE_FORTUNETELLINGOTHER | `custom_send_fortunetellingother(str)` |
| 871 | CLIENT_CUSTOMMSG_ACCEPT_FORTUNETELLING_RESULT | `custom_accept_fortunetelling_result(var)` |
| 873 | CLIENT_CUSTOMMSG_SUPER_BOOK_FACULTY | `request_quit_SBFaculty(int)` |
| 875 | CLIENT_CUSTOMMSG_TIGUANCHALLENGE | `custom_tiguanchallenge(...)` |
| 876 | CLIENT_CUSTOMMSG_DANSHUA_TIGUAN | `custom_send_danshua_tiguan_msg(int,...)` |
| 877 | CLIENT_CUSTOMMSG_CHANGE_GEM_JOB | `custom_change_gem_job(...)` |
| 878 | CLIENT_CUSTOMMSG_ONESTEP_EQUIP_MODIFY_NAME | `custom_send_onestep_modify(...)` |
| 882 | CLIENT_CUSTOMMSG_PET | `custom_pet(int,str,str)` |
| 884 | CLIENT_CUSTOMMSG_QUESTION | `custom_question(int,...)` |
| 885 | CLIENT_CUSTOMMSG_CHECK_AFK_QUESTION | `custom_check_afk_question(int,...)` |
| 886 | CLIENT_CUSTOMMSG_ONESTEP_EQUIP_ADD | `custom_onestep_equip_msg_add(int,int,str)` |
| 887 | CLIENT_CUSTOMMSG_ONESTEP_EQUIP_DEL | `custom_onestep_equip_msg_del(int,int)` |
| 888 | CLIENT_CUSTOMMSG_ONESTEP_EQUIP | `custom_onestep_equip_msg_equipall(int,var)`; `custom_onestep_equip_maintain_tl_set(int,int,str)` |
| 889 | CLIENT_CUSTOMMSG_ONESTEP_EQUIP_CLEAR | `custom_onestep_equip_msg_clear(var,var)` |
| 890 | CLIENT_CUSTOMMSG_NPC_QUERY_ROLE | `custom_trace_role(int)`; `custom_delieve(int,wstr)`; `custom_query_good_feeling(int,str,int)` |
| 893 | CLIENT_CUSTOMMSG_JIANGHU_HOMEBACK | `custom_send_newjh_back_home()` |
| 898 | CLIENT_CUSTOMMSG_NLB_SHIMEN | `custom_send_nlb_shimen(int,...)` |
| 900 | CLIENT_CUSTOMMSG_SONGJING_GAME | `custom_song_jing_request(int)` |
| 901 | CLIENT_CUSTOMMSG_SHIJIA_QTE | `custom_shijia_qte(int)` |
| 902 | CLIENT_CUSTOMMSG_APEX | `custom_apex(int,...)` |
| 903 | CLIENT_CUSTOMMSG_SHIJAI_CHENGJIU | `custom_shijia_achievement_data(int,int)` |
| 904 | CLIENT_CUSTOMMSG_LEARN_GUPU_SKILL | `custom_learn_gupu_skill()` |
| 905 | CLIENT_CUSTOMMSG_GLOBAL_RANK | `send_query_rank(int,str,int)` |
| 906 | CLIENT_CUSTOMMSG_GEAR_PUZZLE | `custom_gear_puzzle(var,var,var)` |
| 908 | CLIENT_CUSTOMMSG_QIN_USESKILL | `custom_qin_use_skill(str)` |
| 909 | CLIENT_CUSTOMMSG_WORD_PUZZLE | `custom_word_puzzle(var,...)` |
| 910 | CLIENT_CUSTOMMSG_IMAGE_PUZZLE | `custom_image_puzzle(var,...)` |
| 911 | CLIENT_CUSTOMMSG_BUSY | `custom_busy(var,...)` |
| 916 | CLIENT_CUSTOMMSG_CLOSE_LIFE_PROMPT | `custom_cancel_open_life_flag(str)` |
| 917 | CLIENT_CUSTOMMSG_CLOSE_LIFE_SKILL_PROMPT | `custom_cancel_open_life_skill_flag(...)` |
| 918 | CLIENT_CUSTOMMSG_GEM_SMAHING_EGG_GAME_MSG | `custom_gem_smahing_egg_game(int,int,int)` |
| 920 | CLIENT_CUSTOMMSG_MAKE_ITEM | `custom_make_item(int,...)` |
| 921 | CLIENT_CUSTOMMSG_JIAHUO | `custom_xjz_jh_game(int,...)` |
| 925 | CLIENT_CUSTOMMSG_BAGUA | `custom_gossip(int,wstr,var)` |
| 931 | CLIENT_CUSTOMMSG_KAPAI_EVENT | `custom_kapai_msg(int,...)` |
| 933 | CLIENT_CUSTOMMSG_EQUIPRANDPROP_LOCK | `custom_msg_recast_lock(int,int,int)` |
| 934 | CLIENT_CUSTOMMSG_EQUIPRANDPROP_UNLOCK | `custom_msg_recast_unlock(int,int,int)` |
| 935 | CLIENT_CUSTOMMSG_YURANDPROP | `custom_msg_recast_randprop(int,int,int,int)` |
| 938 | CLIENT_CUSTOMMSG_SUREEQUIPPROP | `custom_msg_recast_sureprop(int,int,int)` |
| 939 | CLIENT_CUSTOMMSG_CUSTOMEQUIPNAME | `custom_equip_name_start(int,int,int,var,var)` |
| 940 | CLIENT_CUSTOMMSG_GOUHUO | `custom_xjz_gh_skill(int,...)` |
| 941 | CLIENT_CUSTOMMSG_SHAOKAO | `custom_xjz_sk_game(int,...)` |
| 942 | CLIENT_CUSTOMMSG_TREASURE_CHEST | `custom_treasure_chest(int,...)` |
| 943 | CLIENT_CUSTOMMSG_MODIFY_BINGLU | `custom_modify_binglu(int,var,var,var)` |
| 944 | CLIENT_CUSTOMMSG_CHESS_GAME | `custom_chess_game_msg(int,...)` |
| 946 | CLIENT_CUSTOMMSG_CUJU | `custom_cuju(int,...)` |
| 948 | CLIENT_CUSTOMMSG_FORCE_ACTIVITY | `custom_force_act(int,...)` |
| 950 | CLIENT_CUSTOMMSG_COMPLETE_FRESHMAN_HELP_STEP | `custom_help_step_complete(int,str)` |
| 951 | CLIENT_CUSTOMMSG_END_FRESHMAN_HELP | `custom_end_freshman_help()` |
| 952 | CLIENT_CUSTOMMSG_GAME_MATCH | `custom_game_match(int,...)` |
| 953 | CLIENT_CUSTOMMSG_REVENGE | `custom_revenge_match(int,...)` |
| 954 | CLIENT_CUSTOMMSG_WEATHER | `custom_get_area_tianqi(int,var)` |
| 955 | CLIENT_CUSTOMMSG_MATCH_DATA | `custom_query_match_data(int,...)` |
| 956 | CLIENT_CUSTOMMSG_GAMBLE | `custom_request_gamble(int,...)` |
| 957 | CLIENT_CUSTOMMSG_GET_PLAYER_INFO | `custom_send_get_player_game_info(wstr)` |
| 959 | CLIENT_CUSTOMMSG_RECORD_ACCOUNT_LOG | `custom_record_account_log(var)` |
| 962 | CLIENT_CUSTOMMSG_CHANGE_RIDE_SWIM_SPEED | `custom_change_ride_swim_speed(int)` |
| 963 | CLIENT_CUSTOMMSG_USE_LIEZHEN_SKILL | `custom_send_liezhen_skill(int,str)` |
| 964 | CLIENT_CUSTOMMSG_ROB_PRISON | `custom_rob_prison(var,...)` |
| 965 | CLIENT_CUSTOMMSG_HUASHAN | `custom_request_huashan(int,...)`; `custom_wuxue_canwu(int)` |
| 966 | CLIENT_CUSTOMMSG_GROUP_PURCHASE | `custom_group_purchase_msg(int,str)`; `custom_act_grouppurchase(int)`; `custom_act_grouppurchase_data(int)` |
| 967 | CLIENT_CUSTOMMSG_LOTTERY | `custom_lottery_msg(...)` |
| 968 | CLIENT_CUSTOMMSG_HUASHAN_MVP | `custom_get_mvp(int)`; `custom_ballot(int,var,var,var,...)` |
| 969 | CLIENT_CUSTOMMSG_MARRY_SHOP | `custom_msg_marryshop(int,var)` |
| 971 | CLIENT_CUSTOMMSG_ARROW_PUZZLE | `custom_arrow_puzzle(var,...)` |
| 972 | CLIENT_CUSTOMMSG_SINGLE_STEP | `custom_send_single_step(...)` |
| 973 | CLIENT_CUSTOMMSG_GUIDE_ADD_SCHOOL | `custom_guild_add_school(int,...)` |
| 975 | CLIENT_CUSTOMMSG_GET_LEAVE_SCHOOL_INFO | `custom_get_leave_school_info(int,wstr)` |
| 977 | CLIENT_CUSTOMMSG_MANY_REVENGE | `custom_manyrevenge_match(int,wstr,...)`; `custom_manyrevenge_match(int,int,...)` |
| 978 | CLIENT_CUSTOMMSG_QIANJIUCHENG_GAME | `custom_qianjiu_game(int,...)` |
| 980 | CLIENT_CUSTOMMSG_LINK_TO_FLOAT | `custom_send_link_to_float_area(obj,var,var,var,var)` |
| 981 | CLIENT_CUSTOMMSG_LINK_MOVE | `custom_send_link_move(var,var,var,var)` |
| 982 | CLIENT_CUSTOMMSG_UNLINK_FROM_FLOAT | `custom_send_unlink_from_float_area(var,var,var,var)` |
| 984 | CLIENT_CUSTOMMSG_GET_BINGLU_INFO | `custom_send_get_binglu_info(wstr)` |
| 985 | CLIENT_CUSTOMMSG_ACTIVITY_GUIDE | `custom_msg_activity_guide(int)` |
| 988 | CLIENT_CUSTOMMSG_WEATHER_WAR | `custom_weather_war(int,...)`; `custom_weather_war_subgroup(int,...)` |
| 990 | CLIENT_CUSTOMMSG_SYN_NPC_POS | `custom_send_syn_npc_pos(obj,var,var,var)` |
| 991 | CLIENT_CUSTOMMSG_TEAM_FACULTY | `custom_team_faculty(int,var)` |
| 992 | CLIENT_CUSTOMMSG_MARKET | `custom_market_msg(int,var,var,var,var)` |
| 994 | CLIENT_CUSTOMMSG_SCHOOL_CONTRIBUTE | `custom_school_gxd()` |
| 995 | CLIENT_CUSTOMMSG_WORLD_WAR | `custom_cure_hero_npc(int,str,int)`; `custom_world_war_sender(int,...)` |
| 996 | CLIENT_CUSTOMMSG_ARENA | `custom_arena_create(int,int)`; `custom_arena_init(int,...)`; `custom_arena_join(int,int)`; `custom_arena_leave(int)` +3 |
| 997 | CLIENT_CUSTOMMSG_SCHOOL | `custom_send_Request_Declare_Fight(int,int)`; `custom_get_school_msg_info(int,int,int)`; `custom_return_school_home_point(int)`; `custom_get_school_vote_info(int)` +1 |
| 998 | CLIENT_CUSTOMMSG_SCHOOL_FIGHT | `custom_send_request_look_school_fight_info(int)`; `custom_send_request_look_school_fight_info(int,int,int,int)`; `custom_send_request_look_school_fight_info(int,int)`; `custom_send_request_look_school_fight_info(int,int,int)` +18 |
| 999 | CLIENT_CUSTOMMSG_SCHOOL_POSE_FIGHT | `custom_request_school_pose_fight(var,...)` |
| 1000 | CLIENT_CUSTOMMSG_END_PENANCE | `end_school_penance()` |
| 1001 | CLIENT_CUSTOMMSG_SEEK_MINE | `custom_seek_mine_msg(int,int)`; `custom_seek_mine_moveto_msg(int,str,int)` |
| 1002 | CLIENT_CUSTOMMSG_SHIJIA | `custom_shijia_trains(int,int)` |
| 1003 | CLIENT_CUSTOMMSG_MATCH_SANMENG | `custom_match_sanmeng(int,...)` |
| 1004 | CLIENT_CUSTOMMSG_GET_EQUIP_INFO | `custom_send_get_equip_info(wstr)` |
| 1005 | CLIENT_CUSTOMMSG_GET_BASE_INFO | `custom_send_get_base_info(wstr)` |
| 1006 | CLIENT_CUSTOMMSG_FACULTY_INFO | `custom_send_faculty_msg(int,var,var)` |
| 1007 | CLIENT_CUSTOMMSG_FIGHT_ENT_ANI_END | `custom_send_fight_ent_ani_end(obj,str)` |
| 1008 | CLIENT_CUSTOMMSG_GET_DROP_MONEY_ITEM | `custom_pickup_money_item(int)` |
| 1009 | CLIENT_CUSTOMMSG_SCENARIO_BLACK | `custom_scenario_black()` |
| 1010 | CLIENT_CUSTOMMSG_NEW_PACKET | `custom_send_newpacket(int,int,int,int,int,int,int,wstr,str,wstr,wstr,int)`; `custom_get_newpacket(int,str)`; `custom_see_singlepacket(int,str)`; `custom_see_getrecord(int)` +2 |
| 1011 | CLIENT_CUSTOMMSG_NEW_WORLD_BOSS | `request_quit_new_world_boss(int)` |
| 1014 | CLIENT_CUSTOMMSG_GUILD | `custom_request_join_guild(int,var)`; `custom_request_guild_candidate(int,int,int)`; `custom_request_accept_guild_member(int,var)`; `custom_request_reject_guild_member(int,var)` +34 |
| 1015 | CLIENT_CUSTOMMSG_DAY_REDPT/CLIENT_CUSTOMMSG_SCHOOL_WAR | `custom_pick_daypacket(int,...)` |
| 1016 | CLIENT_CUSTOMMSG_GUILDBUILDING | `custom_put_item_from_guilddepot_to_toolbox(int,int,int,int,int,int,var)`; `custom_contribute_money(int,int,var)`; `custom_contribute_goods(int,var,var,var,var)`; `custom_upgrade_faster(int,var,var)` +7 |
| 1017 | CLIENT_CUSTOMMSG_GUILDDOMAIN_OPER | `custom_buy_guild_domain(int,int)`; `custom_guild_domain_page_up_down(int,int)` |
| 1018 | CLIENT_CUSTOMMSG_GUILD_FUNC_OPER | `custom_guild_func_jiguan(var,var,var)`; `custom_guild_viewport_jiguan(int)` |
| 1019 | CLIENT_CUSTOMMSG_VERSION_GUIDE | `custom_version_guide(int)` |
| 1021 | CLIENT_CUSTOMMSG_REMOVE_BUFFER | `custom_remove_buffer(str)` |
| 1023 | CLIENT_CUSTOMMSG_REQ_GUILD_FIRE_TASK | `custom_request_fire_guild_info(int,int,int)`; `custom_request_fire_guild_task(int,wstr,int)`; `custom_request_fire_newstrike_info(int)`; `custom_req_domain_build_info(int,int)` +5 |

### 未解析到 ID 常量的发送调用（子命令直传）

| 发送函数 | ID 表达式 | 参数 |
|---|---|---|
| `custom_get_cross_speaker_data` | CLIENT_CUSTOMMSG_CHAT_MSG | int,... |
| `custom_get_speaker_rank` | CLIENT_CUSTOMMSG_CHAT_MSG | int |
| `custom_gminfo` | CLIENT_CUSTOMMSG_GM | var |
| `custom_update_ng_bind` | CLIENT_CUSTOMMSG_NEIGONGBIND | var |
| `custom_set_zhenyan` | CLIENT_CUSTOMMSG_SET_ZHENYAN | obj |
| `custom_convert_item` | CLIENT_CUSTOMMSG_CONVERT_ITEM | int,... |
| `custom_delete_item` | CLIENT_CUSTOMMSG_DELETEITEM | int,int,int |
| `custom_arrange_item` | CLIENT_CUSTOMMSG_ARANGEITEM | int,int,int |
| `custom_arrange_item_by_type` | CLIENT_CUSTOMMSG_ARANGEITEM | int,int,int,int |
| `custom_condition_details` | CLIENT_CUSTOMMSG_REQUEST_CONDITION_SHOP | int |
| `custom_buy_item` | CLIENT_CUSTOMMSG_BUY_FROM_SHOP | str,int,int,int |
| `custom_sell_item` | CLIENT_CUSTOMMSG_SELL_TO_SHOP | int,int,int,str |
| `custom_open_mount_shop` | CLIENT_CUSTOMMSG_OPEN_MOUNT_SHOP | int |
| `custom_set_shortcut` | CLIENT_CUSTOMMSG_SETSHORTCUT | int,var,var |
| `custom_remove_shortcut` | CLIENT_CUSTOMMSG_REMOVESHORTCUT | int |
| `custom_add_relation` | CLIENT_CUSTOMMSG_RELATION | int,wstr,int,int |
| `custom_show_npc_friend` | CLIENT_CUSTOMMSG_SHOW_NPC_FRIEND |  |
| `custom_kickout_team` | CLIENT_CUSTOMMSG_TEAM_KICKOUT | wstr |
| `custom_leave_team` | CLIENT_CUSTOMMSG_TEAM_LEAVE |  |
| `custom_caption_change` | CLIENT_CUSTOMMSG_TEAM_CHANGE_CAPTAIN | wstr |
| `custom_team_destroy` | CLIENT_CUSTOMMSG_TEAM_DESTROY |  |
| `custom_team_request_join` | CLIENT_CUSTOMMSG_TEAM_REQUEST_JOIN | wstr |
| `custom_team_invite` | CLIENT_CUSTOMMSG_TEAM_INVITE | wstr |
| `custom_change_team_type` | CLIENT_CUSTOMMSG_TEAM_TYPE_SET | int |
| `custom_team_member_set` | CLIENT_CUSTOMMSG_TEAM_MEMBER_SET | int,int,wstr |
| `custom_team_sign_set` | CLIENT_CUSTOMMSG_TEAM_SET_SIGN | str |
| `custom_team_create` | CLIENT_CUSTOMMSG_TEAM_CREATE |  |
| `custom_team_info_refresh` | CLIENT_CUSTOMMSG_TEAM_REFRESH_INFO |  |
| `custom_team_recruit_info_get` | CLIENT_CUSTOMMSG_TEAM_INFO_GET | int,wstr |
| `custom_team_add_info` | CLIENT_CUSTOMMSG_TEAM_ADD_INFO | wstr,wstr,wstr |
| `custom_team_remove_info` | CLIENT_CUSTOMMSG_TEAM_REMOVE_INFO |  |
| `custom_team_player_open` | CLIENT_CUSTOMMSG_TEAM_PLAYER_OPEN |  |
| `custom_team_player_close` | CLIENT_CUSTOMMSG_TEAM_PLAYER_CLOSE |  |
| `custom_set_team_allot_quality` | CLIENT_CUSTOMMSG_TEAM_SET_PICK_QUALITY | int |
| `custom_set_team_allot_mode` | CLIENT_CUSTOMMSG_TEAM_SET_PICK_MODE | int |
| `custom_request_del_appoint_item` | CLIENT_CUSTOMMSG_FESTIVAL_SIGN | int,... |
| `custom_request_festival_sign_info` | CLIENT_CUSTOMMSG_FESTIVAL_SIGN | int,... |
| `custom_request_festival_sign_acctask` | CLIENT_CUSTOMMSG_FESTIVAL_SIGN | int,int |
| `custom_request_festival_sign_get_big` | CLIENT_CUSTOMMSG_FESTIVAL_SIGN | int |
| `custom_request_festival_sign_card_get` | CLIENT_CUSTOMMSG_FESTIVAL_SIGN | int,int |
| `custom_giveup_task` | CLIENT_CUSTOMMSG_TASK_MSG | int,int |
| `send_task_msg` | CLIENT_CUSTOMMSG_TASK_MSG | int,... |
| `custom_refresh_npc_effect` | CLIENT_CUSTOMMSG_REFRESH_TASK_NPC_EFFECT | int,obj |
| `custom_delete_plot` | CLIENT_CUSTOMMSG_DELETE_PLOT | int |
| `custom_use_adventure_item` | CLIENT_CUSTOMMSG_USE_ADVENTURE_ITEM |  |
| `custom_auto_show_prize_box` | CLIENT_CUSTOMMSG_AUTO_PRIZE_BOX | int |
| `custom_publish_collect_task` | CLIENT_CUSTOMMSG_PUBLISH_COLLECT_TASK | wstr,str,int,int,int |
| `custom_open_publish_form` | CLIENT_CUSTOMMSG_OPEN_PUBLISH_FORM |  |
| `custom_open_query_edit_task` | CLIENT_CUSTOMMSG_OPEN_QUERY_EDIT_FORM |  |
| `custom_query_type_task` | CLIENT_CUSTOMMSG_QUERY_TYPE_TASK | int |
| `custom_task_prev_next_page` | CLIENT_CUSTOMMSG_QUERY_PREV_NEXT_PAGE | str,int,int |
| `custom_query_select_edittask_info` | CLIENT_CUSTOMMSG_QUERY_SELECT_TASK_INFO | int |
| `custom_accept_edit_task` | CLIENT_CUSTOMMSG_ACCEPT_EDIT_TASK | int |
| `custom_open_edit_task_accepted` | CLIENT_CUSTOMMSG_OPEN_EDIT_FORM_ACCEPTED |  |
| `custom_open_depot` | CLIENT_CUSTOMMSG_OPEN_DIRECTDEPOT |  |
| `custom_request` | CLIENT_CUSTOMMSG_REQUEST | int,var,... |
| `custom_request_answer` | CLIENT_CUSTOMMSG_REQUEST_ANSWER | int,var,int |
| `custom_exchange_ok` | CLIENT_CUSTOMMSG_EXCHANGE_OK |  |
| `custom_exchange_lock` | CLIENT_CUSTOMMSG_EXCHANGE_LOCK |  |
| `custom_exchange_cancel` | CLIENT_CUSTOMMSG_EXCHANGE_CANCEL | int |
| `custom_exchange_setmoney` | CLIENT_CUSTOMMSG_EXCHANGE_MONEY | var,var |
| `custom_exchange_add_item` | CLIENT_CUSTOMMSG_EXCHANGE_ADD_ITEM | int,int,int,int |
| `custom_exchange_remove_item` | CLIENT_CUSTOMMSG_EXCHANGE_REMOVE_ITEM | int |
| `custom_request_stall` | CLIENT_CUSTOMMSG_REQUEST_STALL |  |
| `custom_stall_begin_stall` | CLIENT_CUSTOMMSG_BEGIN_STALL | wstr,int,wstr |
| `custom_stall_add_item` | src_view | int,int,int,int,int |
| `custom_stall_remove_item` | CLIENT_CUSTOMMSG_REMOVE_STALL_ITEM | int |
| `custom_stall_stop_stall` | CLIENT_CUSTOMMSG_CLOSE_STALL |  |
| `custom_stall_set_itemprice` | CLIENT_CUSTOMMSG_SET_STALL_ITEMPRICE | int,int,int |
| `custom_stall_buy_from_stall` | CLIENT_CUSTOMMSG_OFFLINE_BUY_OTHERONE_STALL | wstr,str,int,i64 |
| `custom_stall_return_ready` | CLIENT_CUSTOMMSG_OFFLINE_CLOSE_STALL |  |
| `custom_open_lookup_stall` | CLIENT_CUSTOMMSG_OPEN_LOOKUP_STALL | wstr |
| `custom_close_lookup_stall` | CLIENT_CUSTOMMSG_CLOSE_LOOKUP_STALL |  |
| `custom_set_purchase_item` | CLIENT_CUSTOMMSG_SET_PURCHASE_ITEM | int,str,int,i64,i64 |
| `custom_del_purchase_item` | CLIENT_CUSTOMMSG_DEL_PURCHASE_ITEM | int |
| `custom_sell_purchase_item` | CLIENT_CUSTOMMSG_SEL_PURCHASE_ITEM | int,int,str,int,i64,i64 |
| `custom_sell_purchase_item_select` | CLIENT_CUSTOMMSG_SEL_PURCHASE_ITEM_SELECT | int,int |
| `custom_open_mail_box` | CLIENT_CUSTOMMSG_POST_OPENBOX | int |
| `custom_send_letter` | CLIENT_CUSTOMMSG_POST_SEND | wstr,wstr,wstr,str,str,str |
| `custom_read_letter` | CLIENT_CUSTOMMSG_POST_READ | str |

## 二、仅定义无 Lua 发送方的 ID（引擎 C++ 直发，需抓包/逆向还原）

共 184 个: 1=CLIENT_CUSTOMMSG_CHAT, 7=CLIENT_CUSTOMMSG_HIRE_SHOP, 12=CLIENT_CUSTOMMSG_STRESS_TEST, 18=CLIENT_CUSTOMMSG_CONSUME_BACK, 21=CLIENT_SUBMSG_REQUIRE_FACTION, 66=CLIENT_CUSTOMMSG_GET_RESEARCH_URL, 82=CLIENT_CUSTOMMSG_MONITER, 85=CLIENT_CUSTOMMSG_FUSE_DAZAOTAI, 106=CLIENT_CUSTOMMSG_LOOK_ITEM, 109=CLIENT_CUSTOMMSG_SET_HOMEPOINT, 117=CLIENT_CUSTOMMSG_CHITCHAT_SEND_CONTENT, 118=CLIENT_CUSTOMMSG_SNS_PALERY_INFO_SEND, 122=CLIENT_CUSTOMMSG_SNS_GIFT, 124=CLIENT_CUSTOMMSG_SNS_ACCPECT_PRESENT, 125=CLIENT_CUSTOMMSG_SNS_LINGQU_PRESENT, 126=CLIENT_CUSTOMMSG_SNS_QIFU, 127=CLIENT_CUSTOMMSG_SNS_ACCPECT_QIFU, 128=CLIENT_CUSTOMMSG_SNS_LINGQU_QIFU, 136=CLIENT_CUSTOMMSG_ACCEPT_HELPER_INVITATION, 137=CLIENT_CUSTOMMSG_REFUSE_HELPER_INVITATION, 138=CLIENT_CUSTOMMSG_ACCEPT_CAPTAIN_RESET_CLONE, 139=CLIENT_CUSTOMMSG_REFUSE_CAPTAIN_RESET_CLONE, 141=CLIENT_CUSTOMMSG_VOTE_OPTION_SELECT, 146=CLIENT_CUSTOMMSG_APPLY_BOSS_HELPER, 147=CLIENT_CUSTOMMSG_CANCEL_BOSS_HELPER, 148=CLIENT_CUSTOMMSG_ALLOW_BOSS_CALL_HELP, 153=CLIENT_CUSTOMMSG_ACCEPT_ADVE_SUMMON, 154=CLIENT_CUSTOMMSG_REFUSE_BOSS_CALL_HELP, 158=CLIENT_CUSTOMMSG_REQUEST_CREATE_AVATAR_CLONE, 159=CLIENT_CUSTOMMSG_GIVEUP_CREATE_AVATAR_CLONE, 161=CLIENT_CUSTOMMSG_JINGMAI, 163=CLIENT_CUSTOMMSG_SNS_NEWS, 171=CLIENT_CUSTOMMSG_QS_HELPER, 179=CLIENT_CUSTOMMSG_MAP_MASK, 203=CLIENT_CUSTOMMSG_BUY_DIRECTDEPOT, 207=CLIENT_CUSTOMMSG_TRANSFER, 211=CLIENT_CUSTOMMSG_USE_SKILL, 212=CLIENT_CUSTOMMSG_RESUME_SKILL, 215=CLIENT_CUSTOMMSG_USE_NEIGONG, 216=CLIENT_CUSTOMMSG_USE_QINGGONG, 217=CLIENT_CUSTOMMSG_QG_SKILL, 218=CLIENT_CUSTOMMSG_ACTIVE_PARRY, 230=CLIENT_CUSTOMMSG_REQ_START_NEIGONG_COMPARE, 234=CLIENT_CUSTOMMSG_CONSIGN_BASE, 237=CLIENT_CUSTOMMSG_NEWSSETTING_MSG, 240=CLIENT_CUSTOMMSG_SITCROSS, 247=CLIENT_CUSTOMMSG_MULTIRIDE_REQUEST, 248=CLIENT_CUSTOMMSG_SABLE_HANDLE, 249=CLIENT_CUSTOMMSG_WIND_HOLD_POWER_END, 260=CLIENT_CUSTOMMSG_USE_ZHENFA, 261=CLIENT_CUSTOMMSG_END_ZHENFA, 270=CLIENT_CUSTOMMSG_USE_ITEMSKILL, 273=CLIENT_CUSTOMMSG_TRAINPAT_PUTBACK, 301=CLIENT_CUSTOMMSG_VIP_MSG, 328=CLIENT_CUSTOMMSG_PROCESS_ADD_ITEM, 329=CLIENT_CUSTOMMSG_PROCESS_REMOVE_ITEM, 354=CLIENT_CUSTOMMSG_POST_GET_MONEY, 355=CLIENT_CUSTOMMSG_POST_GET_ITEM, 371=CLIENT_CUSTOMMSG_JOINTEACHPRNELIST, 372=CLIENT_CUSTOMMSG_EXITTEACHPRNELIST, 376=CLIENT_CUSTOMMSG_QUERYALLPRENTICELIST, 381=CLIENT_CUSTOMMSG_PICKUP_TASK_DROPITEM, 395=CLIENT_CUSTOMMSG_PRESENT_TO_NPC, 406=CLIENT_CUSTOMMSG_ESCORT_SKILL, 420=CLIENT_CUSTOMMSG_RANDOM_SHOP_CLOSE, 434=CLIENT_CUSTOMMSG_ROGUE, 435=CLIENT_CUSTOMMSG_COMPETE_CLONE, 436=CLIENT_CUSTOMMSG_ZODIACTASK, 476=CLIENT_CUSTOMMSG_FLEE_BACHELOR, 478=CLIENT_CUSTOMMSG_TEAM_CONFIRM, 480=CLIENT_CUSTOMMSG_STOPGAME, 481=CLIENT_CUSTOMMSG_CONNECTGAME_RESULT, 482=CLIENT_CUSTOMMSG_MINIGAME_HANDWRITING_RESULT, 483=CLIENT_CUSTOMMSG_FORGEGAME_RESULT, 484=CLIENT_CUSTOMMSG_QINGAME_RESULT, 485=CLIENT_CUSTOMMSG_PICTUREGAME_RESULT, 486=CLIENT_CUSTOMMSG_FORTUNETELLINGGAME_RESULT, 487=CLIENT_CUSTOMMSG_JIIGSAWGAME_RESULT, 488=CLIENT_CUSTOMMSG_WEIQIGAME_RESULT, 489=CLIENT_CUSTOMMSG_FINDPICGAME_RESULT, 490=CLIENT_CUSTOMMSG_PICHAIGAME_RESULT, 491=CLIENT_CUSTOMMSG_SUDOKUGAME_RESULT, 492=CLIENT_CUSTOMMSG_AUGUR_RESULT, 493=CLIENT_CUSTOMMSG_START_FORTUNETELLING, 498=CLIENT_CUSTOMMSG_WQGAME_OPERATE, 500=CLIENT_CUSTOMMSG_RPSGAME_MSG, 506=CLIENT_CUSTOMMSG_MINIGAME_ROPESWING, 507=CLIENT_CUSTOMMSG_GUMU_HZDD, 528=CLIENT_CUSTOMMSG_USE_LUCKITEM, 543=CLIENT_CUSTOMMSG_USE_CHARGEITEM, 548=CLIENT_CUSTOMMSG_SPECIAL_ADV, 551=CLIENT_CUSTOMMSG_CREATE_SCENE_TRANS_TOOL, 557=CLIENT_CUSTOMMSG_CPF_HELPER, 560=CLIENT_CUSTOMMSG_CHARGE_SHOP, 610=CLIENT_CUSTOMMSG_PREPARE_PROCESS, 628=CLIENT_CUSTOMMSG_COMPOUND, 632=CLIENT_CUSTOMMSG_SKILLPAGE, 635=CLIENT_CUSTOMMSG_ALCHEMY_MSG, 650=CLIENT_CUSTOMMSG_REQUEST_OFFLINE_STALL, 651=CLIENT_CUSTOMMSG_OPEN_OFFLINE_SELL, 652=CLIENT_CUSTOMMSG_ADD_OFFLINE_SELL_ITEM, 653=CLIENT_CUSTOMMSG_OFFLINE_REMOVE_SELL_ITEM, 654=CLIENT_CUSTOMMSG_OFFLINE_SET_SELL_ITEMPRICE, 655=CLIENT_CUSTOMMSG_OFFLINE_BEGIN_STALL, 657=CLIENT_CUSTOMMSG_OFFLINE_END_STALL, 659=CLIENT_CUSTOMMSG_OFFLINE_SET_PURCHASE_ITEM, 660=CLIENT_CUSTOMMSG_OFFLINE_DEL_PURCHASE_ITEM, 661=CLIENT_CUSTOMMSG_OFFLINE_SEL_PURCHASE_ITEM, 663=CLIENT_CUSTOMMSG_REMOVE_ALL_SELL, 664=CLIENT_CUSTOMMSG_REMOVE_ALL_BUY, 665=CLIENT_CUSTOMMSG_START_OFFLINE_STALL, 666=CLIENT_CUSTOMMSG_END_REQUEST_STALL, 667=CLIENT_CUSTOMMSG_SELECT_REQUEST_STALL, 668=CLIENT_CUSTOMMSG_STALL_BUY_STYLE, 669=CLIENT_CUSTOMMSG_STALL_POSITION, 670=CLIENT_CUSTOMMSG_LIFE_SERVER_SET, 671=CLIENT_CUSTOMMSG_STALL_LIUYAN, 672=CLIENT_CUSTOMMSG_STALL_LIUYAN_CLEAR, 673=CLIENT_CUSTOMMSG_STALL_JILU_CLEAR, 700=CLIENT_CUSTOMMSG_ACTIVE_ORIGIN, 706=CLIENT_CUSTOMMSG_REQUIRE_NOTIFY_ORIGIN, 710=CLIENT_CUSTOMMSG_CONFIRM, 714=CLIENT_CUSTOMMSG_SKILL_SELECT_LIMIT, 715=CLIENT_CUSTOMMSG_WUXIAN_FETE, 717=CLIENT_CUSTOMMSG_OUTLAND, 772=CLIENT_CUSTOMMSG_CAPTURE_CITY_PRE, 779=CLIENT_CUSTOMMSG_NEW_SCHOOL_TRENDS, 780=CLIENT_CUSTOMMSG_PUZZLE_GAME, 781=CLIENT_CUSTOMMSG_BALANCE_GROW, 782=CLIENT_CUSTOMMSG_BALANCE_WAR, 791=CLIENT_CUSTOMMSG_JZ, 792=CLIENT_CUSTOMMSG_XTB_CONVOY, 809=CLIENT_CUSTOMMSG_START_OFFLINE_JOB, 810=CLIENT_CUSTOMMSG_END_OFFLINE_JOB, 811=CLIENT_CUSTOMMSG_START_OFFLINE_TRUSTEE, 812=CLIENT_CUSTOMMSG_END_OFFLINE_TRUSTEE, 813=CLIENT_CUSTOMMSG_PUSH_BOX_GAME, 814=CLIENT_CUSTOMMSG_OFFLINE_END_TOGETHER, 815=CLIENT_CUSTOMMSG_OFFLINE_TRUSTEE_INTERACT, 816=CLIENT_CUSTOMMSG_OFFLINE_INTERACT, 817=CLIENT_CUSTOMMSG_OFFLINE_GET_FEE_PRIZE, 818=CLIENT_CUSTOMMSG_OFFLINE_GET_DAILY_PRIZE, 819=CLIENT_CUSTOMMSG_ASK_OPEN_DAILY, 820=CLIENT_CUSTOMMSG_OFFLINE_EMPLOY, 822=CLIENT_CUSTOMMSG_OFFLINE_GIVEUP_ABDUCT, 824=CLIENT_CUSTOMMSG_GET_OFFLINE_PRIZE, 830=CLIENT_CUSTOMMSG_OFFLINE_STOP_HELP, 832=CLIENT_CUSTOMMSG_CALL_HELPER, 834=CLIENT_CUSTOMMSG_FIND_WATER, 836=CLIENT_CUSTOMMSG_RECEIVE_LOG_MSG, 858=CLIENT_CUSTOMMSG_NEW_TERRITORY, 859=CLIENT_CUSTOMMSG_BEGIN_PRACTICEQIN, 872=CLIENT_CUSTOMMSG_SEARCH_XUECHOU, 883=CLIENT_CUSTOMMSG_YY_TASK, 891=CLIENT_CUSTOMMSG_LANTERN, 892=CLIENT_CUSTOMMSG_JIANGHU_GOUHUO, 894=CLIENT_CUSTOMMSG_JIANGHU_MEET, 895=CLIENT_CUSTOMMSG_FRESHMAN_JIAYUAN, 896=CLIENT_CUSTOMMSG_DOUBLE_XIULIAN, 897=CLIENT_CUSTOMMSG_CHANGFENG_MOVETO, 907=CLIENT_CUSTOMMSG_SHIJIA_YANWU, 914=CLIENT_CUSTOMMSG_ASY_POS_IN_ZHAOSHI_END, 922=CLIENT_CUSTOMMSG_GEM_RESET_SKILL, 923=CLIENT_CUSTOMMSG_GEM_GAME_MSG, 924=CLIENT_CUSTOMMSG_MSG_DELAY, 926=CLIENT_CUSTOMMSG_ARREST_WARRANT, 927=CLIENT_CUSTOMMSG_DARE_SCHOOL_TASK, 928=CLIENT_CUSTOMMSG_NETBAR_GET_RPIZE, 929=CLIENT_CUSTOMMSG_MOBILE_COMFIRM, 930=CLIENT_CUSTOMMSG_GIFT_COMFIRM, 932=CLIENT_CUSTOMMSG_THIRD_TASK, 936=CLIENT_CUSTOMMSG_YUDESIGNATEPROP, 937=CLIENT_CUSTOMMSG_RESUMEEQUIPPROP, 945=CLIENT_CUSTOMMSG_CHESS_RULE, 974=CLITENT_CUSTOMMSG_TALENT_SYSTEM, 979=CLIENT_CUSTOMMSG_CONTINUE_ENTRY_SCENE, 983=CLIENT_CUSTOMMSG_JPIRATES_ATTACK, 986=CLIENT_CUSTOMMSG_NEW_WAR_RULE, 987=CLIENT_CUSTOMMSG_DREAM_LAND, 989=CLIENT_CUSTOMMSG_GAME_PAIJIU, 993=CLIENT_CUSTOMMSG_PUZZLE, 1012=CLIENT_CUSTOMMSG_STOP_BIND, 1013=CLIENT_CUSTOMMSG_USE_NPC_SKILL, 1020=CLIENT_CUSTOMMSG_AS_TIME

## 三、服务端→客户端 handler 函数名册（665 个，命令名寻址）

| 函数名 | msg_type 后形参 |
|---|---|
| `console_log` | (无) |
| `custom_handler_init` | (无) |
| `custom_receive_record_account_log` | ... |
| `on_new_player_guid` | transfer_type,... |
| `custom_specific_sysinfo` | tips_type,string_id,... |
| `explain_info` | (无) |
| `prompt_is_valid` | (无) |
| `custom_sysinfo` | tips_type,string_id,... |
| `custom_asyn_time` | server_time |
| `custom_beginmenu` | (无) |
| `custom_addtitle` | ... |
| `custom_addmenu` | ... |
| `custom_endmenu` | target |
| `custom_closemenu` | (无) |
| `custom_begin_curse` | ticks,curse_text,cursetype |
| `custom_end_curse` | msg |
| `open_task_not_complete_info` | task_id,npc,rec_id |
| `change_tasknpc_flag` | npc,flagtype |
| `change_tasknpc_effect` | npc,effecttype |
| `show_share_task_menu` | task_id,task_title,sharer_name |
| `show_query_edit_task_info` | ... |
| `show_select_edittask_info` | task_type,require_info,prop_info,prize_info |
| `open_submit_collect_box` | task_id |
| `custom_be_request` | request_type,request_name,... |
| `send_letter_result` | result |
| `is_delete_letter` | del_type,text_type,serial_no |
| `get_letter_money_success` | serialno |
| `virtualrec_add` | recname,... |
| `virtualrec_clear` | (无) |
| `virtualrec_update` | row,col,value |
| `virtualrec_updaterow` | row,... |
| `virtualrec_removerow` | row |
| `custom_marriage_npc` | flag |
| `custom_neishangcure_npc` | (无) |
| `custom_push_roll_item` | item,item_info,amount,wait_time |
| `custom_team_assign_dlg` | mode |
| `custom_open_compete_baseprice_dlg` | ... |
| `custom_begin_compete` | ... |
| `custom_handle_compete_result_info` | ... |
| `custom_show_animation` | name,index |
| `custom_del_animation` | name |
| `buy_stall_style_ok` | (无) |
| `open_tool_box` | (无) |
| `custom_new_born` | page |
| `custom_play_point_sound` | sound_name,x,y,z |
| `custom_play_gui_sound` | sound_name |
| `on_npc_talk` | talk_obj,talk_id,stage,mode,configid,name,other_info_1,other_info_2,other_info_3 |
| `random_adventure_system_info` | text_id,hour,minute,second,npc_id,npc |
| `on_npc_karma_spring` | npc,old_val,new_val |
| `on_show_npc_friend` | ... |
| `on_update_npc_karma_value` | npc,karma |
| `on_update_npc_karma` | npc_id,karma |
| `custom_receive_item_info` | player_name,item_info,uniqueid |
| `custom_game_stump_start` | time |
| `custom_game_stump_stop` | (无) |
| `custom_refresh_origin` | ... |
| `custom_notify_a_new_origin` | ... |
| `custom_shenghuo_action` | obj_id,flag |
| `custom_can_get_side_origin` | origin_id,ret |
| `custom_have_offline_prize` | (无) |
| `custorm_player_enter_scene_onready` | scene,first |
| `custorm_player_oncontinue` | scene |
| `custom_show_model_dialog_info` | string_id,... |
| `custom_life_skill_hit` | flowname,result |
| `custom_zhenfa_line` | target,line_type |
| `custom_zhenfa_map` | target,op_type |
| `custom_refresh_zhenfa_effect` | type |
| `custom_set_city_show` | city_name,b_first |
| `custom_offline_not_finish` | (无) |
| `custom_show_offline_job_log` | (无) |
| `custom_show_offline_job_daily` | date,show_type,job_id1,job_id2,job_id3,job_id4,job_id5,idCnt,sceneID,jobSuff,isNewRole,stage |
| `custom_hide_offline_job_daily` | (无) |
| `custom_show_offline_prize` | date,offTime,prizeCnt,... |
| `custom_show_offline_job_interact` | job_id,fee_list,cvt_flag |
| `custom_hide_offline_job_interact` | (无) |
| `custom_show_offline_job_interact_prize` | prize_type,item_id,num,money_type,money,name,desc |
| `custom_hide_offline_job_interact_prize` | (无) |
| `custom_show_offline_togather` | ratio |
| `custom_close_client` | (无) |
| `captain_reset_clone` | captain_name,scene_configid,level |
| `custom_update_player_pos` | player,x,y,z,o |
| `custom_get_a_new_lifejob` | jobid |
| `custom_open_help_form` | taskid |
| `custom_offlinework_online_effect` | target,... |
| `custom_get_a_new_skill` | skillid,skilltype |
| `custom_show_life_freshmanhelp` | noticetype |
| `time_limit_show` | cur_time |
| `custom_help_step` | stepID |
| `custom_start_inter_action` | player,npc,player_action,npc_action |
| `custom_end_inter_action` | player,npc |
| `custom_band_relive_pos` | player |
| `custom_relive_end` | player,relive_type |
| `custom_relive_check` | ret |
| `custom_donghai_relive_check` | ret |
| `custom_forge_started` | (无) |
| `custom_forge_stopped` | (无) |
| `custom_forge_timeout` | (无) |
| `custom_forge_cancel_step` | id |
| `custom_forge_step_diff` | index,diff |
| `hide_npc_client` | ident |
| `custom_show_roadsign` | npc |
| `custom_show_base_info` | ... |
| `custom_show_binglu_info` | ... |
| `custom_show_equip_info` | ... |
| `custom_show_game_info` | ... |
| `start_augur` | (无) |
| `end_augur` | index |
| `on_clf_rollgame` | ... |
| `on_hwq_choice` | ... |
| `on_gumu_hzdd` | ... |
| `on_wuxian_fete` | ... |
| `custom_can_fortunetellingstall` | (无) |
| `custom_start_fortunetellingstall` | text,stallname |
| `stall_pathing_okay` | (无) |
| `close_business` | (无) |
| `stall_position_okay` | stall_pos_index |
| `liuyan_okay` | (无) |
| `liuyan_clear_okay` | (无) |
| `jilu_clear_okay` | (无) |
| `custom_canceled_fortunetellingstall` | (无) |
| `custom_need_fortunetelling` | target,spritemoney,personmoney,godmoney,rank |
| `custom_accpet_fortunetelling_result` | desc,money |
| `custom_xuechou_result` | msg_id,... |
| `close_gmcc_form` | msg_id,... |
| `custom_can_stall` | (无) |
| `can_select_stall_ok` | stall_type,scene_id,x,y,z,o |
| `custom_goto` | player,doornpcid |
| `custom_stall_okay` | nOnLine |
| `custom_stall_canceled` | (无) |
| `can_offline_stall` | (无) |
| `custom_offline_stall_okay` | text |
| `npc_begin_pause_skill` | npc,time |
| `npc_end_pause_skill` | npc |
| `change_crop_grow_effect` | crop |
| `npc_fight_animatio_start` | ani_name,npc,state |
| `custom_faculty_msg` | arg1,arg2 |
| `custom_team_faculty` | sub_cmd,... |
| `custom_shool_faculty` | sub_cmd,... |
| `custom_arena` | sub_cmd,... |
| `on_open_arena_query` | arg1,arg2 |
| `custom_holdfaculty_msg` | sub_cmd,arg1 |
| `custom_trigger_skill_effect` | attacker,skillID,... |
| `custom_refresh_track_info` | first_flag,server_time |
| `show_hide_npc` | npc |
| `custom_game_pichai_start` | (无) |
| `custom_game_pichai_stop` | (无) |
| `skill_icon_flash` | index,flag |
| `play_music` | flag,music |
| `school_pose_fight` | submsg_type,... |
| `school_fight` | submsg_type,... |
| `gather_effect` | npc,effect_type,slice,effect_name,effect_pos,x,y,z |
| `task_msg_logic` | submsg,... |
| `ask_descapital_addbuff` | submsg,... |
| `out_clone_time` | time |
| `open_reset_clone_ui` | ... |
| `avatar_clone_notice` | (无) |
| `avatar_clone_helper` | (无) |
| `invite_helper` | invitor_name |
| `avatar_clone_invite_new_help` | (无) |
| `on_client_tips` | subMsg,info,return_info |
| `custom_show_clone_menu` | npcid,cloneid,... |
| `custom_close_clone_menu` | ... |
| `custom_inphase_clone` | uid |
| `custom_clone_lead_form_open` | col_id |
| `custom_clone_lead_form_close` | col_id |
| `custom_clone_lead_form_change` | col_id,spring_id,spring_flg |
| `custom_self_add_form` | (无) |
| `custom_random_clone_enter_clone` | (无) |
| `custom_close_self_add_form` | (无) |
| `custom_wuxue_flash` | (无) |
| `drive_skill_open` | npc,open_flag |
| `refresh_trainpat_btn` | sign |
| `on_custom_guild_enemy` | ... |
| `show_buy_guild_domain_form` | sub_msg,... |
| `show_hide_drop_view` | open_type |
| `start_world_trans` | open_type |
| `end_world_trans` | open_type |
| `start_scene_trans` | ... |
| `end_scene_trans` | open_type |
| `publish_news` | newsType,repeatTime,infoId,... |
| `show_time_limit_form` | (无) |
| `close_time_limit_form` | (无) |
| `show_count_limit_form` | form_name,form_count,form_title,form_info |
| `inc_count_limit_form` | form_name,count_value |
| `close_count_limit_form` | (无) |
| `notify_desc_limit_form` | form_type,limit_type,desc |
| `clear_desc_limit_form` | form_type,limit_type |
| `asyn_server_time` | server_time |
| `show_stall_config` | (无) |
| `custom_world_rank` | ... |
| `custom_begin_breath_in_water` | (无) |
| `custom_end_breath_in_water` | (无) |
| `asyn_update_head_text` | pause_time |
| `custom_update_head_text` | name,str_id,delaytime,pause_time |
| `custom_fishing_form` | (无) |
| `showform_offline_onhand` | (无) |
| `game_minigame_itemresult` | ... |
| `begin_life_trade` | share_type,... |
| `camera_vibrate` | vibrate_name |
| `stop_use_skill` | (无) |
| `stall_success_shengji` | ntype |
| `on_stall_note` | note_type,... |
| `show_vote_form` | vote_id |
| `show_vote_result` | vote_id,... |
| `on_use_luckitem` | name |
| `recv_buy_guild_domain_data` | msgid,... |
| `tiguan_all_msg` | type,sub_type,... |
| `tiguanchallenge_msg` | sub_type,... |
| `battlefield_all_msg` | sub_type,... |
| `custom_producejiguan_show` | sub_msg,... |
| `begplayer_result` | target,rank |
| `on_gem_temp_skill_set` | ... |
| `on_hire_shop_msg` | sub_msg,... |
| `on_point_consign_msg` | ... |
| `on_del_tl` | ... |
| `on_prepare_fortunetellingother_result` | target,rank |
| `on_charge_shop_msg` | ... |
| `on_npc_personnate` | ... |
| `on_give_item` | npc,... |
| `on_give_clone_award` | award_desc,drop_id,exp_1,use_time |
| `on_end_clone_award` | ... |
| `on_guild_bank_show` | ... |
| `on_spy_msg` | ... |
| `install_tmp_skill_item` | sub_msg,... |
| `uninstall_tmp_skill_item` | sub_msg,... |
| `close_tmp_skill_item` | sub_msg,... |
| `on_huihai_msg` | sub_cmd,... |
| `on_fuse_begin_msg` | job_id,... |
| `on_home_msg` | sub_cmd,... |
| `on_skillpage_fuse_begin_msg` | npcid,... |
| `on_compound_begin_msg` | job_id,... |
| `on_split_begin_msg` | job_id,... |
| `on_npc_exchange_begin_msg` | job_id,... |
| `on_changeequip_begin_msg` | job_id,... |
| `on_contribute_begin_msg` | job_id,... |
| `on_npcexchange_begin_msg` | npc_id,... |
| `on_upgrade_equip_msg` | ... |
| `on_convert_equip_msg` | ... |
| `on_qimen_equip_msg` | ... |
| `on_fire_guild_msg` | sub_cmd,... |
| `on_all_escort_list_msg` | ... |
| `on_escort_info_msg` | escort_info,... |
| `on_escort_open_form_msg` | npcname,escort_list |
| `on_escort_begin_msg` | ... |
| `on_escort_audio_msg` | sound_name |
| `on_escort_end_msg` | ... |
| `on_server_escort_time_begin` | ... |
| `on_server_escort_time_end` | ... |
| `on_server_escort_msg` | ... |
| `on_select_bindtype` | str_info,str_var |
| `on_split_item_form_open` | job_id |
| `on_split_item_break_btn_show` | show,job_id |
| `on_fuse_item_break_btn_show` | show |
| `on_split_item_success` | job_id |
| `on_open_play_animation_form` | animation,isautostop |
| `on_close_play_animation_form` | (无) |
| `on_open_job_form` | num,... |
| `process_single_msg` | sub_type,value |
| `on_start_chinese_chess_game` | server_time |
| `on_escort_list_open` | escortnpc,ntype |
| `on_escort_isspecialtime` | nIsTime,nMsgID |
| `on_form_livegroove_to_faculty` | (无) |
| `on_leitai_ready_confirm` | ... |
| `on_begin_roll_game` | lpoint,rpoint |
| `on_vip_msg` | sub_cmd,... |
| `on_server_leitai_call_for_confirm` | time |
| `on_server_leitai_syn_time` | status,time |
| `on_server_leitai_wager_info` | npc_id |
| `on_server_leitai_reward_info` | npc_id |
| `on_server_leitai_reward_count` | reward_num |
| `on_server_leitai_qiecuo_count` | qiecuo_num |
| `on_server_leitai_integral_info` | integral_info |
| `on_server_leitai_war_info` | ... |
| `on_server_clone_challenge_success_record` | ... |
| `on_sns_search_friend_result` | ... |
| `on_sns_enemy_info_result` | ... |
| `on_sns_player_info_result` | ... |
| `on_sns_send_present_result` | bSuccuss |
| `on_chitchat_request_chat_result` | ... |
| `on_chitchat_get_player_info` | ... |
| `custom_activity_notice` | ... |
| `on_systemchat_black_auction` | ... |
| `on_groupchat_send_content_result` | ... |
| `on_chitchat_send_group_content_result` | ... |
| `on_sns_accpect_present_result` | bSuccuss |
| `on_sns_send_okey_result` | optype |
| `on_sns_accpect_qifu_result` | bSuccuss |
| `on_sns_send_qifu_result` | bSuccuss |
| `on_server_sns_msg_sysinfo` | feed_type,name,feed_id,feed_function,text,... |
| `on_server_tvt_info` | (无) |
| `on_server_sns_msg_feed` | ... |
| `show_guild_girl_form` | ... |
| `custom_show_relation_add_apply` | name,relation,relation_old |
| `on_custom_interact_msg` | sub_type,... |
| `on_custom_open_schoolwar_dead_form` | sceneid,... |
| `on_custom_updata_worldwar_publicpoint` | ... |
| `on_bribe_play_game_msg` | ... |
| `on_home_point_msg` | sub_type,... |
| `on_recive_newneigongpk_msg` | ... |
| `on_recive_gmcc_msg` | ... |
| `on_custom_show_public_msgbox` | ... |
| `on_custom_show_public_inputbox` | ... |
| `on_custom_school_msg` | ... |
| `on_custom_school_homepoint_msg` | ... |
| `on_custom_unenthrall_msg` | ... |
| `on_pop_repair_form` | ... |
| `on_pop_repair_addbag` | ... |
| `on_open_fight_form` | arg_1,arg_2 |
| `on_update_free_fight_rank` | ... |
| `on_end_free_fight_rank` | ... |
| `begin_task_1201` | Repute,npc_ident,... |
| `on_open_shortcut_key_form` | (无) |
| `on_npc_head_talking` | talk_obj,talk_id,... |
| `on_open_helper_show` | ... |
| `on_close_leitai_time_panel` | ... |
| `on_buffer_change` | type,buffer_id,owner,sender,owner_name,sender_name,... |
| `on_show_sole_prompt` | val1,val2 |
| `on_auto_equip_shotweapon` | val1 |
| `on_show_faction` | ... |
| `on_open_xietie_msg` | ... |
| `on_recive_systemfriends_msg` | ... |
| `on_recive_schoolpose_vote` | ... |
| `on_recive_schoolpose_prompt` | ... |
| `on_recive_ruselt_gifttonpc_msg` | ... |
| `on_table_info` | ownername,table_remaintime,item_name,compose_remaintime,... |
| `on_arrest_warrant` | ... |
| `on_school_vote_info` | ... |
| `on_refresh_client_npc` | client_npc_file |
| `on_zhenfa_liezhen` | ... |
| `on_boss_call_help` | boss_configid,boss_help_info |
| `on_cpf_helper` | ... |
| `on_qs_helper` | player_name,help_info |
| `on_open_freshman_help` | ... |
| `on_parser_customizing` | ... |
| `on_server_defend_temple` | ... |
| `on_freshman_open_help_info` | ... |
| `on_buy_captal_msg` | ... |
| `on_fight_error_sysinfo` | error_index,... |
| `on_open_clone_col_award` | score_level,ntotal_score,col_use_time,col_try_count |
| `on_boss_strength_change` | boss_id,... |
| `custom_market_msg` | sub_cmd,arg1,arg2,arg3,arg4,arg5 |
| `on_custom_request_sure_dialog_model` | tvt_type,string_id,... |
| `on_custom_rob_prison` | sub_type,... |
| `on_dare_school_task` | submsg,... |
| `on_check_second_word` | ... |
| `on_begin_drama` | ... |
| `on_minigame_wq_is_look` | player |
| `on_open_single_limit_form` | ... |
| `on_close_single_limit_form` | ... |
| `on_adve_summon_request` | adve_name,group_id |
| `on_mobile_comfirm` | ... |
| `on_open_prize_code` | ... |
| `on_enthrall_special_time_msg` | ... |
| `on_keep_item` | ... |
| `on_open_school_introduce` | type |
| `on_join_school_movie` | cartoon_id |
| `on_npc_show` | NpcType |
| `on_open_equip_blend` | opt |
| `on_open_gift_code` | ... |
| `on_open_yytask_code` | ... |
| `on_process_pet_sable` | ... |
| `on_open_sable_sikll` | ... |
| `on_yufeng_hold_power` | ... |
| `on_normal_pet_msg` | sub_msg_id,... |
| `on_kapai_msg` | ... |
| `on_open_check_switch_enable` | switch_type,enable |
| `custom_open_holiday_active` | ... |
| `on_jyz_sever_msg_handle` | ... |
| `on_jyz_high_light_btn` | ... |
| `on_jyz_map_icon_state` | ... |
| `on_trigger_debug_show` | ... |
| `on_query_fresh_man` | ... |
| `on_alchemy_msg` | ... |
| `on_treasurebox_msg` | ... |
| `on_use_tonic` | (无) |
| `custom_jingmai_msg` | sub_cmd,... |
| `on_gamble_msg` | ... |
| `custom_box_event` | ... |
| `custom_treasure_chest` | ... |
| `custom_new_composite` | ... |
| `custom_avenge` | ... |
| `on_teacher_pupil_msg` | sub_type,... |
| `custom_gongxun_msg` | gxd,... |
| `custom_clonestore_openlight` | ... |
| `custom_clonestore_receive` | ... |
| `custom_clonestore_selled` | ... |
| `custom_clonestore_recooling` | ... |
| `custom_clonestore_punish` | ... |
| `custom_jianghu_boss_msg` | ... |
| `custom_jiuyinzhi_current_step_msg` | ... |
| `custom_jzsj_qhyc` | ... |
| `custom_ssg_schoolmeet` | ... |
| `custom_wxj_schoolmeet` | ... |
| `custom_dmp_schoolmeet` | ... |
| `custom_clone_head` | ... |
| `on_custom_group_purchase` | ... |
| `custom_single_set_form` | value |
| `on_custom_seek_mine` | ... |
| `on_custom_lottery_info` | ... |
| `custom_show_schoolleader_info` | ... |
| `on_activity_value_rank_prize_msg` | ... |
| `on_huashan_msg` | ... |
| `custom_qinggong_use_failed` | obj_id,qinggong_type |
| `on_custom_get_power_redeem_num` | sub_msg,num |
| `on_custom_get_novice_redeem_num` | num |
| `on_custom_get_old_redeem_num` | num |
| `custom_recastattribute_randprop_open` | open_id |
| `custom_npc_query_role` | sub_cmd,... |
| `custom_recastattribute_randprop_succeed` | op_type,newpackprop |
| `custom_recastattribute_sure` | op_type |
| `custom_marry` | sub_type,... |
| `on_present_to_npc` | ... |
| `custom_single_player_module_info` | sub_type,... |
| `on_plate_award` | server_cmd_id,module,... |
| `open_name_modify_ui` | ... |
| `custom_reconnect_server` | ip_addr,port,server_id,scene_id,cross_type,cross_server_type |
| `custom_reconnect_noraml_server` | ip_addr,port,server_id,scene_id,cross_target_type,cross_server_type,player_cross_name,player_customizing,player_prompt_freshman_info,player_sole_prompt |
| `custom_reconnect_new_server` | ip_addr,port,server_id,scene_id,cross_target_type |
| `on_dongfang_msg` | sub_cmd,... |
| `on_modify_binglu_msg` | item_id,... |
| `on_egg_get_item` | ... |
| `on_star_show` | ... |
| `on_show_form` | (无) |
| `on_weather_sky` | sub_type,... |
| `on_revenge_match` | sub_type,... |
| `on_server_custommsg_match_data` | sub_type,... |
| `on_whack_info` | sub_type,... |
| `on_egwar_msg` | ... |
| `on_msg_day_divine` | ... |
| `on_register_msg` | ... |
| `on_match_msgbox` | ... |
| `on_condition_details` | ... |
| `on_make_item` | msgid,... |
| `on_weather_war_msg` | submsg_type,... |
| `on_extra_skill_msg` | ... |
| `on_copy_skill_msg` | ... |
| `on_taohua_maze` | ... |
| `on_custom_package_destroyed` | (无) |
| `on_force_activity` | ... |
| `on_new_school_trends` | ... |
| `on_resetface_succeed` | op_type |
| `on_custom_open_disguise` | open_type |
| `on_rpg_game_msg` | ... |
| `on_custom_equip_name` | open |
| `on_custom_equip_name_succ` | (无) |
| `on_custom_pikongzhang_activity` | sub_type,... |
| `on_custom_show_tersuer_max_value` | ... |
| `on_custom_show_tersuer_result` | sub_type,... |
| `on_custom_update_tersure_upgrade` | (无) |
| `on_custom_update_tersure_add_taolu` | (无) |
| `on_custom_tersure_nor_add_taolu` | (无) |
| `on_custom_update_tersure_restore_taolu` | (无) |
| `on_custom_show_tersure` | sub_type |
| `on_custom_itemexchange` | msgid,... |
| `on_xjz_gouhuo` | ... |
| `on_xjz_jiahuo` | ... |
| `on_musical_note` | value |
| `on_xjz_shaokao` | ... |
| `on_custom_open_check_jj_form` | msgid,... |
| `on_custom_round_task` | ... |
| `on_set_player_camera` | ... |
| `on_open_check_player_form` | ... |
| `on_leave_school` | leavetype,... |
| `on_open_world_boss_demage_stat` | sub_type,... |
| `on_open_nlb_shimen_form` | sub_type,... |
| `on_open_sjy_shimen_form` | sub_type,... |
| `on_open_xmg_shimen_form` | sub_type,... |
| `on_open_tyhg_shimen_form` | sub_type,... |
| `on_open_whz_shimen_form` | sub_type,... |
| `on_repent_msg` | sub_type,... |
| `on_close_school_introduce_form` | sub_type,... |
| `on_get_leave_school_info` | sub_type,name,info |
| `is_new_school` | (无) |
| `is_force` | (无) |
| `simulate_close_scene` | ... |
| `get_origin_succeed` | origin_id |
| `on_addbtn_lead_info_open` | (无) |
| `on_interactive_furniture` | sub_type,... |
| `on_random_question_msg` | ... |
| `on_question_msg` | sub_type,... |
| `on_check_afk_question_msg` | ... |
| `on_push_box` | sub_type,... |
| `on_inv_rank_info` | ... |
| `on_depot_msg` | ... |
| `on_home_depot_msg` | ... |
| `on_home_gouhuo_msg` | ... |
| `on_trans_server_list` | ... |
| `custom_accost` | ... |
| `custom_sweet_employ` | ... |
| `on_sweet_pet_equip` | ... |
| `on_flash_window` | (无) |
| `on_activity_partner` | ... |
| `on_custom_begin_scene_compete` | ... |
| `on_consume_back_msg` | ... |
| `on_use_lantern_msg` | sub_type,... |
| `on_bodyguardoffice_msg` | sub_type,... |
| `on_open_shop_exchange_form` | ... |
| `on_call_helper_msg` | sub_type,... |
| `on_open_swap_goods_form` | ... |
| `get_scene_id_by_name` | (无) |
| `on_open_scene_jhpk` | sub_type,... |
| `on_wuji_helper` | ... |
| `on_jianghu_gouhuo_msg` | ... |
| `on_card_msg` | ... |
| `on_find_water_msg` | ... |
| `open_new_bag` | sub_type,... |
| `on_jianghu_award_msg` | ... |
| `oldbag_to_newbag1` | ... |
| `on_form_interface` | file,func,... |
| `on_sworn` | sub_msg,... |
| `on_ancienttomb` | sub_msg,... |
| `on_tg_roll_game` | sub_msg,... |
| `on_world_auction` | sub_msg,... |
| `on_agree_war` | sub_msg,... |
| `on_outland_war` | sub_msg,... |
| `on_server_custommsg_puzzle_game` | (无) |
| `on_server_custommsg_jh_explore` | sub_type,... |
| `learn_jingmai` | ... |
| `on_jianghu_guide` | ... |
| `on_cross_school_fight` | ... |
| `on_gumu_rescue_msg` | ... |
| `on_double_xiulian_msg` | sub_msg,... |
| `on_accept_friend_letter_flag` | ... |
| `on_changfeng_moveto` | ... |
| `on_guild_station_msg` | ... |
| `on_guild_city_def_msg` | ... |
| `on_job_msg` | submsg,... |
| `on_dbomall_msg` | sub_msg,... |
| `on_festival_sign` | sub_msg,... |
| `on_five_sky` | sub_msg,... |
| `on_ghost_city` | sub_msg,... |
| `on_useitem_open_appoint_form` | ... |
| `on_draw_prize` | sub_msg,... |
| `on_huashan_jianbei_msg` | ... |
| `on_many_revenge` | sub_type,... |
| `on_new_territory_msg` | ... |
| `on_body_change` | ntype,action_id |
| `on_weather_war_switch` | ... |
| `on_statue_msg` | sub_msg,... |
| `on_skill_select_limit` | sub_msg,... |
| `on_change_body_fuse` | sub_msg,... |
| `on_fuse_dazaotai` | sub_msg,... |
| `on_custom_wujidao` | sub_msg,... |
| `on_redpacket_msg` | ... |
| `on_linxiao_zhizun` | sub_msg,... |
| `on_custom_songjing_game` | sub_msg,... |
| `on_custom_newtresure_update` | sub_type |
| `on_custom_refresh_newtresure_upgrade` | (无) |
| `on_msg_outland_event` | sub_msg,... |
| `on_open_binglu_recover_form` | ... |
| `on_msg_bad_guy_vally_info` | sub_msg,... |
| `on_msg_activity_guide_info` | sub_msg,... |
| `on_sky_hill_info` | sub_msg,... |
| `on_san_hill_info` | sub_msg,... |
| `on_single_shop_info` | sub_msg,... |
| `on_msg_shijia_achievement_info` | sub_msg,... |
| `on_school_extinct` | sub_msg,... |
| `on_msg_shijia_trans` | sub_msg,times |
| `on_msg_shijia_play_time` | ... |
| `on_msg_yanmen_bazhentu` | ... |
| `on_msg_match_sanmeng` | ... |
| `on_super_market_msg` | sub_msg,... |
| `on_msg_auction` | sub_msg,... |
| `on_msg_shijia_yanwu` | sub_msg,... |
| `on_guild_luck_pray` | sub_msg,... |
| `on_new_teacher_msg` | sub_msg,... |
| `on_select_school_confirm` | ... |
| `on_school_contribute_limit_msg` | ... |
| `on_betray_school_confirm` | ... |
| `on_royal_treasure` | sub_msg,... |
| `on_balance_grow_msg` | sub_msg,... |
| `on_balance_war_msg` | sub_msg,... |
| `on_wudao_msg` | sub_msg,... |
| `on_luandou_msg` | sub_msg,... |
| `on_taosha_msg` | sub_msg,... |
| `on_apex_msg` | sub_msg,... |
| `on_see_msg` | sub_msg,... |
| `custom_gupu_learn` | ... |
| `on_sns_news` | sub_msg,... |
| `on_home_farm` | sub_msg,... |
| `on_home_mine` | sub_msg,... |
| `on_handle_jyf` | sub_msg,... |
| `on_delay_show_jyf_info` | (无) |
| `on_custom_sign_n_draw` | sub_msg,... |
| `on_custom_online_purchase_rebate` | sub_msg,... |
| `on_custom_guildglobalwar` | sub_msg,... |
| `on_cross_station_war_msg` | sub_msg,... |
| `on_cross_server_msg` | sub_msg,... |
| `on_league_matches_msg` | sub_msg,... |
| `on_flee_in_night_msg` | sub_msg,... |
| `on_assemble_stone_msg` | sub_msg,... |
| `on_kof_msg` | sub_msg,... |
| `on_gew_msg` | sub_msg,... |
| `on_custom_gwd` | sub_msg,... |
| `on_custom_ine` | sub_msg,... |
| `on_custom_myr` | sub_msg,... |
| `on_custom_shm` | sub_msg,... |
| `on_custom_gpw` | sub_msg,... |
| `on_sec_msg` | sub_msg,... |
| `on_champion_war_msg` | sub_msg,... |
| `on_over_cloud_msg` | sub_msg,... |
| `on_custom_ghw` | sub_msg,... |
| `on_custom_qfj` | sub_msg,... |
| `on_custom_mojue` | sub_msg,... |
| `on_custom_itt` | sub_msg,... |
| `on_custom_cem` | sub_msg,... |
| `on_custom_gxb` | sub_msg,... |
| `on_custom_slt` | sub_msg,... |
| `on_custom_cuh` | sub_msg,... |
| `on_challenge_clone_msg` | sub_msg,... |
| `on_tarot_msg` | sub_msg,... |
| `custom_rogue` | sub_msg,... |
| `custom_zodiac` | sub_msg,... |
| `on_wudao_yanwu_msg` | sub_msg,... |
| `on_activity_horse_race_msg` | sub_msg,... |
| `on_custom_check_delete_force` | ... |
| `on_tj_redpacket` | submsg,... |
| `on_custom_newpacket` | submsg,... |
| `on_custom_attribute_mall` | submsg,... |
| `on_custom_ver_201904` | submsg,... |
| `on_msg_new_war_rule` | ... |
| `on_msg_dream_land` | ... |
| `on_msg_get_achievement_points` | ... |
| `on_msg_chess_game` | sub_msg,... |
| `on_msg_chess_rule` | ... |
| `on_msg_hide_puzzles` | submsg,... |
| `on_msg_compete_clone` | submsg,... |
| `on_leitai_compete` | submsg,talk_obj,talk_id,... |
| `on_qianjiu_war_msg` | submsg,... |
| `on_custom_qianjiucheng_game` | submsg,... |
| `on_custom_version_guide` | submsg,... |
| `on_custom_gear_puzzle` | submsg,... |
| `on_custom_cross_invade` | submsg,... |
| `on_custom_global_rank` | submsg,... |
| `on_custom_wanyingjingtian_price` | sub_msg,... |
| `on_custom_school_xtb_convoy` | submsg,... |
| `on_custom_school` | submsg,... |
| `on_custom_bag_assistant` | ... |
| `on_custom_common_compete` | submsg,... |
| `custom_fight_statistics` | submsg,... |
| `custom_collect_equip` | submsg,... |
| `on_custom_common_wuxue_canwu` | submsg,... |
| `on_custom_annotate_fresh` | submsg,... |
| `on_custom_new_world_boss` | ... |
| `on_custom_island_plunder` | submsg,... |
| `on_custom_word_puzzle` | submsg,... |
| `on_custom_jpirates_attack` | submsg,... |
| `on_custom_jinwu_stealth` | submsg,... |
| `on_custom_challenge_hill` | submsg,... |
| `on_custom_home_workshop` | submsg,... |
| `on_custom_jjgg` | sub_msg,... |
| `on_custom_balance_pk` | sub_msg,... |
| `on_custom_sandsea_convoy` | sub_msg,... |
| `on_custom_image_puzzle` | submsg,... |
| `on_custom_painting_depot` | sub_msg,... |
| `on_custom_cross_recruit` | sub_msg,... |
| `on_custom_xialuxiangfeng` | ... |
| `on_custom_new_ride` | sub_msg,... |
| `on_custom_cuju` | sub_msg,... |
| `on_custom_newschool_shimen` | sub_type,... |
| `on_custom_mobile_id` | sub_msg,... |
| `on_custom_zlcl_plunder` | sub_msg,... |
| `on_custom_cross_zlcl` | sub_type,... |
| `on_custom_busy_feedback` | sub_type,... |
| `on_custom_arrow_puzzle` | submsg,... |
