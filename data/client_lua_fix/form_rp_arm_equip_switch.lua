-- =============================================================
-- 角色属性面板-外观切换功能（江湖装/身份装/时装）
-- 需要添加到 form_stage_main\form_role_info\form_rp_arm.lua 中
-- =============================================================

-- 1. 在 form_rp_arm_init 函数中初始化 show_equip_type
function form_rp_arm_init(form)
  form.show_body = false
  form.Fixed = true
  form.body_actor2 = nx_null()
  form.show_equip_type = 0
  -- 初始化3个单选按钮的可见性
  form.rbtn_jianghu.Visible = true
  form.rbtn_menpai.Visible = true
  form.rbtn_shizhuang.Visible = true
end

-- 2. 修复后的显示切换按钮函数
function on_show_equip_btns(form)
  if not nx_is_valid(form) then
    return
  end
  -- 先全部隐藏
  form.btn_jianghu.Visible = false
  form.btn_shizhuang.Visible = false
  form.btn_menpai.Visible = false

  -- 根据当前显示的装备类型，显示"切换到其他类型"的按钮
  if form.show_equip_type == 0 then
    -- 当前是江湖装：显示切换到时装和身份装的按钮
    form.btn_shizhuang.Visible = true
    form.btn_menpai.Visible = true
  elseif form.show_equip_type == 1 then
    -- 当前是身份装：显示切换到江湖装和时装的按钮
    form.btn_jianghu.Visible = true
    form.btn_shizhuang.Visible = true
  elseif form.show_equip_type == 2 then
    -- 当前是时装：显示切换到江湖装和身份装的按钮
    form.btn_jianghu.Visible = true
    form.btn_menpai.Visible = true
  end
end

-- 3. 3个单选按钮的点击事件
-- 江湖装单选按钮
function on_rbtn_jianghu_checked_changed(rbtn)
  if rbtn.Checked then
    local form = rbtn.ParentForm
    form.show_equip_type = 0
    nx_execute("custom_sender", "custom_showequip_type", 0)
    on_show_equip_btns(form)
    -- 刷新角色模型显示
    refresh_role_scenebox()
  end
end

-- 身份装单选按钮
function on_rbtn_menpai_checked_changed(rbtn)
  if rbtn.Checked then
    local form = rbtn.ParentForm
    form.show_equip_type = 1
    nx_execute("custom_sender", "custom_showequip_type", 1)
    on_show_equip_btns(form)
    -- 刷新角色模型显示
    refresh_role_scenebox()
  end
end

-- 时装单选按钮
function on_rbtn_shizhuang_checked_changed(rbtn)
  if rbtn.Checked then
    local form = rbtn.ParentForm
    form.show_equip_type = 2
    nx_execute("custom_sender", "custom_showequip_type", 2)
    on_show_equip_btns(form)
    -- 刷新角色模型显示
    refresh_role_scenebox()
  end
end

-- 4. 3个切换按钮的点击事件（点击切换到对应的装备类型）
-- 点击"江湖装"按钮
function on_btn_jianghu_click(btn)
  local form = btn.ParentForm
  if not is_body_changing() then
    form.rbtn_jianghu.Checked = true
  end
end

-- 点击"时装"按钮
function on_btn_shizhuang_click(btn)
  local form = btn.ParentForm
  if not is_body_changing() then
    form.rbtn_shizhuang.Checked = true
  end
end

-- 点击"身份装"按钮
function on_btn_menpai_click(btn)
  local form = btn.ParentForm
  if not is_body_changing() then
    form.rbtn_menpai.Checked = true
  end
end

-- 5. 在 form_rp_arm_open 中读取服务端属性并设置按钮状态
-- 在 form_rp_arm_open 函数中添加以下代码（在设置完 equip_grid 后）：
--[[
  local showequip_type = 0
  local game_client = nx_value("game_client")
  local client_player = game_client:GetPlayer()
  if not nx_is_valid(client_player) then
    return
  end
  if client_player:FindProp("ShowEquipType") then
    showequip_type = client_player:QueryProp("ShowEquipType")
  end
  form.show_equip_type = showequip_type
  if showequip_type == 0 then
    form.rbtn_jianghu.Checked = true
  elseif showequip_type == 1 then
    form.rbtn_menpai.Checked = true
  elseif showequip_type == 2 then
    form.rbtn_shizhuang.Checked = true
  end
  on_show_equip_btns(form)
]]--

-- 6. 在 form_rp_arm_showrole 中刷新模型时传入 show_equip_type
-- 找到 form_rp_arm_showrole 函数中的 role_composite:RefreshBodyNorEquip 调用
-- 将其改为：
--[[
  local showequip_type = 0
  if client_player:FindProp("ShowEquipType") then
    showequip_type = client_player:QueryProp("ShowEquipType")
  end
  role_composite:RefreshBodyNorEquip(client_player, form.actor2, showequip_type)
]]--