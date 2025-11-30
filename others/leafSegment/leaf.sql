CREATE TABLE `leaf_alloc` (
  `biz_tag` varchar(128)  NOT NULL DEFAULT '', 
  `max_id` bigint(20) NOT NULL DEFAULT '1', 
  `step` int(11) NOT NULL, 
  `description` varchar(256)  DEFAULT NULL,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`biz_tag`)
) ENGINE=InnoDB;

-- 初始化一条测试数据
INSERT INTO leaf_alloc (biz_tag, max_id, step, description) VALUES ('order-service', 1, 1000, '订单ID生成');