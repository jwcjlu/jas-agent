-- 迁移脚本：移除知识库表的 agent_id 字段
-- 执行前请备份数据库！

-- 1. 删除外键约束
ALTER TABLE `knowledge_bases` DROP FOREIGN KEY IF EXISTS `knowledge_bases_ibfk_1`;

-- 2. 删除 agent_id 索引
ALTER TABLE `knowledge_bases` DROP INDEX IF EXISTS `idx_agent_id`;

-- 3. 删除 agent_id 字段
ALTER TABLE `knowledge_bases` DROP COLUMN IF EXISTS `agent_id`;

