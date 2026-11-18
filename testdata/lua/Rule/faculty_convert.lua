---****************************************************************************
-- FILENAME	: Rule\\faculty_convert.lua
-- COPYRIGHT: Snail Game
-- AUTHOR	: Mrkk
-- DATE		: 2010年3月1日
-------------------------------------------------------------------------------
-- DESC		: 关于 使用修转化为武学经验 的处理逻辑
---****************************************************************************

-------------------------------------------------------------------------------
-- name	:  convert ( self, convert_type, name, value )
-- desc	:  扣除修为值，增加相应的武学经验值
-- 参数	:  self（玩家）， convert_type（消耗类型 int），
--		   name（武学具体名称）， value（消耗修为值）
-------------------------------------------------------------------------------
local STARTCONVERT_RESULT_SUCCEED = 1                -- 成功
local STARTCONVERT_RESULT_ERROR_NG_MUST_LIMIT = 10   -- 内功必要限制
local STARTCONVERT_RESULT_ERROR_SK_MUST_LIMIT = 11   -- 技能必要限制
local STARTCONVERT_RESULT_ERROR_QG_MUST_LIMIT = 12   -- 轻功必要限制


-- 修为流程结束时调用
function convert_wuxue_exp(self, convert_type, name)
	-- 检测条件
	local result = nx_can_convert(self, convert_type, name)

	if tonumber(result) == STARTCONVERT_RESULT_SUCCEED then

		--获得需要扣除的修为点数,本次转化的经验值
		local value_table = nx_get_convert_value( self, convert_type, name )
		local decValue = nx_int(-tonumber(value_table[1])) -- 消耗的修为值
		local addValue = nx_int(value_table[2])            -- 增加的经验值
		
		--转化为内功经验
		if convert_type == 1 then
			nx_add_neigong_exp(self, name, addValue)
			nx_inc_faculty(self, decValue, "to add neigong exp: " .. name)
		end
		--转化为招式经验
		if convert_type == 2 then

			local skill = nx_find_skill(self, name)
			if not skill or not nx_exists(skill) then
				return
			end

			local cur_exp = nx_int64(nx_query_property(skill, "Exp"))
			local upgrade_exp = nx_int64(nx_query_property(skill, "UpgradeExp"))

			-- 上次条件技能升级条件不足
			if nx_int64_compare_int64(cur_exp, upgrade_exp) then
				nx_add_skill_exp(self, name, 1)
			else
				-- 如果再加一次经验就升级了 则判断是否超过最高升级经验
				-- 如果超过了升级经验则 加与升级经验相差的经验值 按百分比扣除修为值
				if nx_int64_large_int64(nx_int64_inc_int64(cur_exp, nx_int64(addValue)), upgrade_exp) then
					local real_add_value = nx_int(nx_int64_dec_int64(upgrade_exp, cur_exp))

					if real_add_value < nx_int(0) then
						return
					end

					local persent = tonumber(real_add_value) / tonumber(addValue)
					if persent < 0 or persent > 1 then
						return
					end

					local real_dec_value = nx_int(tonumber(decValue) * tonumber(persent))
					if real_dec_value > nx_int(0) then
						return
					end

					nx_add_skill_exp(self, name, real_add_value)
					nx_inc_faculty(self, real_dec_value, "to add zhaoshi exp: " .. name)
				else
					nx_add_skill_exp(self, name, addValue)
					nx_inc_faculty(self, decValue, "to add zhaoshi exp: " .. name)
				end
			end

		end
		--转化为轻功技能经验
		if convert_type == 3 then
			--轻功技能对象
			local qgskill = nx_find_qgskill(self, name)
			if not qgskill or not nx_exists(qgskill) then
				return
			end
		
			--当前经验,升级经验
			local cur_exp = nx_int(nx_query_property(qgskill, "Exp"))
			local upgrade_exp = nx_int(nx_query_property(qgskill, "UpgradeExp"))
			
			--合法性
			if nx_int(cur_exp) > nx_int(upgrade_exp) or nx_int(upgrade_exp) < 1 then
				return
			end

			--当前经验等于升级经验,原因是上次升级未满足条件,直接加1点经验尝试刷新
			if nx_int(cur_exp) == nx_int(upgrade_exp) then
				nx_add_qgskill_exp(self, name, 1)
				return
			end

			--当前经验加上本次增加的经验后,超过升级要求的经验,则计算实际应该增加的经验和扣除的修为
			if nx_int(cur_exp) + nx_int(addValue) > nx_int(upgrade_exp) then
				--到升级实际需要的经验
				local real_add_value = upgrade_exp - cur_exp

				--需要减少的实际修为
				local real_dec_value = nx_int(real_add_value)*-1
				
				--实际属性更新
				nx_add_qinggong_exp( self, name, real_add_value )
				nx_inc_faculty( self, real_dec_value, "to add qinggong exp: " .. name)

			else
				--当前经验加上本次增加的经验后,不足或等于升级要求的经验,直接增加经验
				nx_add_qgskill_exp( self, name, addValue )
				nx_inc_faculty( self, decValue, "to add qinggong exp: " .. name)
			end
		end
	end



end




-- 开始修为回调
function start_convert(self, convert_type, name)
	-- 开始流程时先进入 运功状态
	-- 然后播放融入动作
	-- 当融入动作结束后 玩家会自动返回运功状态
	--nx_set_property(self, "State", "ng_yg_cs")
end

-- 打断修为回调
function break_convert(self, convert_type, name)

end





