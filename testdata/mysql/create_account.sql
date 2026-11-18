DELIMITER $$

DROP PROCEDURE IF EXISTS `nx_district`.`create_account`$$
CREATE DEFINER=`sa`@`%` PROCEDURE  `nx_district`.`create_account`()
BEGIN
declare i int;
set i = 1;
while i < 101 do
insert into dc_accounts values  (concat('test',lpad(i,'3','0')),'test',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'',CURRENT_TIMESTAMP,51,1,11,0,0,0,'');
set i = i + 1;
end while;
END $$

DELIMITER ;