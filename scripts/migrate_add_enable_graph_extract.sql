ALTER TABLE `documents`
    ADD COLUMN `enable_graph_extract` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否提取知识图谱' AFTER `metadata`;

