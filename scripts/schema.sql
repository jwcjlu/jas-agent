-- JAS Agent 数据库表结构

-- Agent 配置表
CREATE TABLE IF NOT EXISTS `agents` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(100) NOT NULL UNIQUE COMMENT 'Agent名称',
  `framework` VARCHAR(20) NOT NULL COMMENT 'Agent框架类型: react, plan, chain, sql, elasticsearch',
  `description` TEXT COMMENT 'Agent描述',
  `system_prompt` TEXT COMMENT '系统提示词',
  `max_steps` INT DEFAULT 10 COMMENT '最大执行步数',
  `model` VARCHAR(50) DEFAULT 'gpt-3.5-turbo' COMMENT '默认使用的模型',
  `config` JSON COMMENT '其他配置（JSON格式）',
  `connection_config` JSON COMMENT '连接配置（MySQL/ES等）',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
  INDEX `idx_framework` (`framework`),
  INDEX `idx_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent配置表';

-- MCP 服务表
CREATE TABLE IF NOT EXISTS `mcp_services` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(100) NOT NULL UNIQUE COMMENT 'MCP服务名称',
  `endpoint` VARCHAR(500) NOT NULL COMMENT 'MCP服务端点URL',
  `description` TEXT COMMENT '服务描述',
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
  `tool_count` INT DEFAULT 0 COMMENT '工具数量',
  `last_refresh` TIMESTAMP NULL COMMENT '最后刷新时间',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='MCP服务表';

-- Agent-MCP 关联表
CREATE TABLE IF NOT EXISTS `agent_mcp_bindings` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `agent_id` INT NOT NULL COMMENT 'Agent ID',
  `mcp_service_id` INT NOT NULL COMMENT 'MCP服务ID',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`agent_id`) REFERENCES `agents`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`mcp_service_id`) REFERENCES `mcp_services`(`id`) ON DELETE CASCADE,
  UNIQUE KEY `uk_agent_mcp` (`agent_id`, `mcp_service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent-MCP绑定表';

-- 插入一些示例数据
INSERT INTO `agents` (`name`, `framework`, `description`, `system_prompt`, `max_steps`, `model`, `connection_config`) VALUES
('默认助手', 'react', '通用智能助手，适合大多数场景', NULL, 10, 'gpt-3.5-turbo', NULL),
('数据分析师', 'plan', '专业的数据分析助手，擅长复杂多步骤任务', '你是一个专业的数据分析师，擅长分解和分析复杂问题。', 20, 'gpt-4', NULL),
('SQL查询助手', 'sql', 'MySQL数据库查询专家', NULL, 15, 'gpt-3.5-turbo', '{"host":"localhost","port":3306,"database":"testdb","username":"root","password":""}'),
('日志分析助手', 'elasticsearch', 'Elasticsearch日志搜索和分析专家', NULL, 15, 'gpt-3.5-turbo', '{"host":"http://localhost:9200","username":"","password":""}');

