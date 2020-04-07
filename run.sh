#!/bin/bash
echo “Running testsuite $1 with additional parameters from command line  $2”
#"loops=1000,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.${add}:3306, verbose=true,printConnectionTime=true, summary=false,reportCSV=true" 
java -Xms2G -Xmx3G -classpath "./*:./lib/*" net.tc.testsuite.$1 "$2" 
