CREATE TABLE IF NOT EXISTS `cgnat_table` (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `private_ip` VARBINARY(16) NOT NULL,
    `public_ip` VARBINARY(16) NOT NULL,
    `start_port` MEDIUMINT NOT NULL,
    `end_port` MEDIUMINT NOT NULL
);

ALTER TABLE `cgnat_table`
    ADD UNIQUE `cgnat_table_private_ip_unique` (`private_ip`);

CREATE TABLE IF NOT EXISTS `whitelist_table` (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `msisdn` VARCHAR(20) NOT NULL
);

ALTER TABLE `whitelist_table`
    ADD UNIQUE `whitelist_table_msisdn_unique` (`msisdn`);


CREATE TABLE IF NOT EXISTS `user_table` (
    `id` TINYINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `username` VARCHAR(50) NOT NULL,
    `email` VARCHAR(70) NULL,
    `password_hash` VARCHAR(255) NOT NULL,
    `user_type` ENUM('admin', 'viewer', 'whitelist') NOT NULL DEFAULT 'viewer',
    `status` BOOLEAN NOT NULL DEFAULT 1,
    `last_login` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE `user_table`
    ADD UNIQUE `user_table_username_unique` (`username`);

ALTER TABLE `user_table`
    ADD UNIQUE `user_table_email_unique` (`email`);


CREATE TABLE IF NOT EXISTS `session_table` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `session_id` VARCHAR(50) NOT NULL,
    `msisdn` VARCHAR(20) NOT NULL,
    `site` VARCHAR(20) NOT NULL,
    `private_ip` VARBINARY(16) NOT NULL,
    `public_ip` VARBINARY(16) NULL,
    `ipv6` VARCHAR(40) NULL,
    `start_port` MEDIUMINT NULL,
    `end_port` MEDIUMINT NULL,
    `packets` MEDIUMINT NOT NULL,
    `wl_status` BOOLEAN NOT NULL,
    `start_time` DATETIME NOT NULL,
    `end_time` DATETIME NOT NULL,

    PRIMARY KEY (`id`, `end_time`),
    UNIQUE KEY `session_table_session_id_unique` (`session_id`, `end_time`),
    INDEX `session_table_msisdn_index` (`msisdn`),
    INDEX `session_table_start_time_index` (`start_time`),
    INDEX `session_table_end_time_index` (`end_time`)
)
PARTITION BY RANGE COLUMNS (`end_time`) (
    PARTITION pmax VALUES LESS THAN (MAXVALUE)
);


CREATE TABLE IF NOT EXISTS `alarm_table` (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `alarm_id` VARCHAR(50) NOT NULL,
    `block_id` VARCHAR(50) NOT NULL,
    `site` VARCHAR(20) NOT NULL,
    `time_raised` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `severity` ENUM('critical', 'high', 'normal', 'low') NOT NULL DEFAULT 'normal',
    `Module` ENUM('network', 'system', 'application') NOT NULL DEFAULT 'network',
    `message` VARCHAR(255) NULL,
    `status` ENUM('new', 'resolved') NOT NULL DEFAULT 'new',
    `updated_by` TINYINT NOT NULL
);

ALTER TABLE `alarm_table`
    ADD INDEX `alarm_table_time_raised_index` (`time_raised`);

ALTER TABLE `alarm_table`
    ADD CONSTRAINT `alarm_table_updated_by_foreign`
    FOREIGN KEY (`updated_by`) REFERENCES `user_table` (`id`);