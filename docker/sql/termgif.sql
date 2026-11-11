DROP DATABASE IF EXISTS `termgif`;
CREATE DATABASE `termgif`
    DEFAULT CHARACTER SET utf8
    DEFAULT COLLATE utf8_general_ci;

USE `termgif`;

DROP TABLE IF EXISTS `gif`;
CREATE TABLE `gif`
(
    id SMALLINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    api_id VARCHAR(255) NOT NULL,
    KEY api_id (`api_id`),
    created  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE = INNODB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `gif_data`;

CREATE TABLE `gif_data` (
                      id MEDIUMINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
                      frame_nb SMALLINT UNSIGNED NOT NULL,
                      delay SMALLINT UNSIGNED NOT NULL,
                      frame MEDIUMTEXT NOT NULL,
                      gif_id VARCHAR(255) NOT NULL,
                      CONSTRAINT `fk_gif_data_gif`
                          FOREIGN KEY (gif_id) REFERENCES gif (api_id)
                              ON DELETE CASCADE
                              ON UPDATE RESTRICT
) ENGINE = InnoDB DEFAULT CHARSET=utf8;