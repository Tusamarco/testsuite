HOW to USE it
===========

`IE: java -Xms2G -Xmx3G -classpath "./*:./lib/*" net.tc.testsuite.MySQLConnectionTest  --help`

`java -Xms2G -Xmx3G -classpath "./*:./lib/*" net.tc.testsuite.MySQLConnectionTest 'loops=10,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306, verbose=true, summary=true'`

or use the run.sh script and add parameters like

`./run.sh "loops=200,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306,sleep=1000,summary=true"`

```******************************************
DB Parameters to use
Parameters are COMMA separated and the whole set must be pass as string
IE java -Xms2G -Xmx3G -classpath "./*:./lib/*" net.tc.testsuite.MySQLConnectionTest "loops=10,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306" 
url [url=jdbc:mysql://127.0.0.1:3306]
user [user=test_user]
password [password=test_pw]
parameters [parameters=&useSSL=false&autoReconnect=true]
schema [schema=test]

*****************************************
Application Parameters 
loops [loops=50
sleep [sleep=0]
verbose [verbose=false]
summary [summary=false]
reportCSV [reportCSV=false]
****************************************
 Optional selectForceAutocommitOff [selectForceAutocommitOff=true]

****************************************
 Optional For the test printConnectionTime [printConnectionTime=true]


```
