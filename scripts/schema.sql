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

-- 知识库表
CREATE TABLE IF NOT EXISTS `knowledge_bases` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(100) NOT NULL COMMENT '知识库名称',
  `description` TEXT COMMENT '知识库描述',
  `tags` JSON COMMENT '标签列表（JSON数组）',
  `embedding_model` VARCHAR(50) DEFAULT 'text-embedding-3-small' COMMENT '使用的嵌入模型',
  `chunk_size` INT DEFAULT 800 COMMENT '文档分块大小',
  `chunk_overlap` INT DEFAULT 120 COMMENT '文档分块重叠大小',
  `vector_store_type` VARCHAR(20) DEFAULT 'memory' COMMENT '向量存储类型: memory, milvus',
  `vector_store_config` JSON COMMENT '向量存储配置（JSON格式）',
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_active` (`is_active`),
  INDEX `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识库表';

-- 文档表
CREATE TABLE IF NOT EXISTS `documents` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `knowledge_base_id` INT NOT NULL COMMENT '所属知识库ID',
  `name` VARCHAR(255) NOT NULL COMMENT '文档名称',
  `file_path` VARCHAR(500) COMMENT '文件路径',
  `file_size` BIGINT COMMENT '文件大小（字节）',
  `file_type` VARCHAR(50) COMMENT '文件类型: pdf, txt, docx, etc.',
  `status` VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending, processing, completed, failed',
  `chunk_count` INT DEFAULT 0 COMMENT '文档分块数量',
  `processed_at` TIMESTAMP NULL COMMENT '处理完成时间',
  `error_message` TEXT COMMENT '错误信息',
  `metadata` JSON COMMENT '文档元数据（JSON格式）',
  `enable_graph_extract` BOOLEAN DEFAULT FALSE COMMENT '是否提取知识图谱',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (`knowledge_base_id`) REFERENCES `knowledge_bases`(`id`) ON DELETE CASCADE,
  INDEX `idx_knowledge_base_id` (`knowledge_base_id`),
  INDEX `idx_status` (`status`),
  INDEX `idx_file_type` (`file_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文档表';

-- 插入一些示例数据
INSERT INTO `agents` (`name`, `framework`, `description`, `system_prompt`, `max_steps`, `model`, `connection_config`) VALUES
('默认助手', 'react', '通用智能助手，适合大多数场景', NULL, 10, 'gpt-3.5-turbo', NULL),
('数据分析师', 'plan', '专业的数据分析助手，擅长复杂多步骤任务', '你是一个专业的数据分析师，擅长分解和分析复杂问题。', 20, 'gpt-4', NULL),
('SQL查询助手', 'sql', 'MySQL数据库查询专家', NULL, 15, 'gpt-3.5-turbo', '{"host":"localhost","port":3306,"database":"testdb","username":"root","password":""}'),
('日志分析助手', 'elasticsearch', 'Elasticsearch日志搜索和分析专家', NULL, 15, 'gpt-3.5-turbo', '{"host":"http://localhost:9200","username":"","password":""}');

