-- PayGate-Omni PostgreSQL 初始化脚本
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
GRANT ALL PRIVILEGES ON DATABASE paygate_omni TO paygate;
