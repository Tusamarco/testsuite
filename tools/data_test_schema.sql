-- MySQL dump 10.13  Distrib 8.0.22, for Linux (x86_64)
--
-- Host: localhost    Database: bobo
-- ------------------------------------------------------
-- Server version	8.0.22

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
SET @MYSQLDUMP_TEMP_LOG_BIN = @@SESSION.SQL_LOG_BIN;
SET @@SESSION.SQL_LOG_BIN= 0;

--
-- GTID state at the beginning of the backup 
--

SET @@GLOBAL.GTID_PURGED=/*!80000 '+'*/ '3248e476-8e7c-11e8-92b9-08002734ed50:1-1101000';

--
-- Table structure for table `address_`
--

DROP TABLE IF EXISTS `address_`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `address_` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `active` tinyint unsigned NOT NULL DEFAULT (0),
  `street` varchar(100) NOT NULL,
  `city` varchar(100) NOT NULL,
  `region` varchar(100) NOT NULL,
  `country_iso3` char(3) NOT NULL,
  `link_id` int unsigned NOT NULL,
  `cityId` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `cities`
--

DROP TABLE IF EXISTS `cities`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cities` (
  `name` varchar(100) COLLATE utf8_bin NOT NULL,
  `country` varchar(50) COLLATE utf8_bin NOT NULL,
  `subcountry` varchar(100) COLLATE utf8_bin DEFAULT NULL,
  `geonameid` int unsigned DEFAULT NULL,
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `country_iso3` char(3) COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=23019 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `continents`
--

DROP TABLE IF EXISTS `continents`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `continents` (
  `iso2` char(2) NOT NULL,
  `iso3` char(3) DEFAULT NULL,
  `name` varchar(50) NOT NULL,
  PRIMARY KEY (`iso2`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `countries`
--

DROP TABLE IF EXISTS `countries`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `countries` (
  `official_name_en` varchar(200) COLLATE utf8mb4_bin NOT NULL,
  `iso2` char(2) COLLATE utf8mb4_bin DEFAULT NULL,
  `iso3` char(3) COLLATE utf8mb4_bin NOT NULL,
  `isonumeric` int DEFAULT NULL,
  `currency_alphabetic_code` char(10) COLLATE utf8mb4_bin DEFAULT NULL,
  `currency_name` varchar(50) COLLATE utf8mb4_bin DEFAULT NULL,
  `english_formal` varchar(200) COLLATE utf8mb4_bin DEFAULT NULL,
  `english_short` varchar(200) COLLATE utf8mb4_bin DEFAULT NULL,
  `cldr_display_name` varchar(200) COLLATE utf8mb4_bin DEFAULT NULL,
  `capital` varchar(200) COLLATE utf8mb4_bin DEFAULT NULL,
  `continent` varchar(200) COLLATE utf8mb4_bin DEFAULT NULL,
  `ds` varchar(10) COLLATE utf8mb4_bin DEFAULT NULL,
  `developed_developing_countries` varchar(10) COLLATE utf8mb4_bin DEFAULT NULL,
  `intermediate_region_code` varchar(20) COLLATE utf8mb4_bin DEFAULT NULL,
  `intermediate_region_name` varchar(20) COLLATE utf8mb4_bin DEFAULT NULL,
  `languages` varchar(100) COLLATE utf8mb4_bin DEFAULT NULL,
  `region_code` int DEFAULT '0',
  `region_name` varchar(50) COLLATE utf8mb4_bin DEFAULT NULL,
  `sub_region_code` int DEFAULT '0',
  `sub_region_name` varchar(50) COLLATE utf8mb4_bin DEFAULT NULL,
  `tld` char(5) COLLATE utf8mb4_bin DEFAULT NULL,
  `is_independent` varchar(50) COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`iso3`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `firstname`
--

DROP TABLE IF EXISTS `firstname`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `firstname` (
  `name` varchar(50) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `gender` tinyint DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=81058 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `lastname`
--

DROP TABLE IF EXISTS `lastname`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `lastname` (
  `name` varchar(50) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=43053 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `population`
--

DROP TABLE IF EXISTS `population`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `population` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `countryname` varchar(200) COLLATE utf8_bin NOT NULL,
  `iso3` char(3) COLLATE utf8_bin DEFAULT NULL,
  `seriesname` varchar(50) COLLATE utf8_bin NOT NULL,
  `seriescode` varchar(50) COLLATE utf8_bin NOT NULL,
  `y2019` bigint unsigned DEFAULT NULL,
  `y2020` bigint unsigned DEFAULT NULL,
  `y2021` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1296 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `streets`
--

DROP TABLE IF EXISTS `streets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `streets` (
  `id` int NOT NULL AUTO_INCREMENT,
  `streetname` varchar(100) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=99716 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `titles`
--

DROP TABLE IF EXISTS `titles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `titles` (
  `name` varchar(50) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `bho` int unsigned DEFAULT NULL,
  `onlyman` int unsigned DEFAULT NULL,
  `alsofem` int unsigned DEFAULT NULL,
  `norder` int unsigned DEFAULT NULL,
  `id` int unsigned NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users_`
--

DROP TABLE IF EXISTS `users_`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users_` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL,
  `lastname` varchar(50) NOT NULL,
  `age` smallint unsigned NOT NULL,
  `phone` varchar(50) NOT NULL,
  `registration_date` datetime NOT NULL,
  `address_id` int unsigned NOT NULL,
  `email` varchar(150) NOT NULL,
  `active` tinyint unsigned NOT NULL DEFAULT (0),
  `gender` char(1) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
SET @@SESSION.SQL_LOG_BIN = @MYSQLDUMP_TEMP_LOG_BIN;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2020-11-09 10:40:21
