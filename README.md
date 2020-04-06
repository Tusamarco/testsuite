# What is it

This test suite is a collection of classes that can be call to perform some of the most basic and often used functionalities tests.
Currently implemented :
  * Connectivity test 
  * stale read test

## HOW to USE it

To run the tests you need to have a Java VM in place (1.8 or newer).
Then just invoke the class name with  --help to see the possible options:  

`IE: java -Xms2G -Xmx3G -classpath "./*:./lib/*" net.tc.testsuite.MySQLConnectionTest  --help`

`java -Xms2G -Xmx3G -classpath "./*:./lib/*" net.tc.testsuite.MySQLConnectionTest 'loops=10,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306, verbose=true, summary=true'`


or use the run.sh script and add class name and  parameters like

`./run.sh MySQLConnectionTest "loops=200,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306,sleep=1000,summary=true"`

## Parameters
Parameters are divided by, connection parameters and application parameters.
 
Some parameters are common to all tests like:


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
```
Some other can be specific to that test

```****************************************
 Optional For the test printConnectionTime [printConnectionTime=true]
```

### Some running example
MySQLConnectionTest : `./run.sh MySQLConnectionTest "loops=100, url=jdbc:mysql://192.168.4.22:3306, verbose=true,summary=true,parameters=&characterEncoding=UTF-8,printConnectionTime=true,reportCSV=false,sleep=1000 help"`
StaleReadTest       : `./run.sh StaleReadTest "loops=10000, url=jdbc:mysql://192.168.4.22:3306, urlRead=jdbc:mysql://192.168.4.23:3306, verbose=true,summary=true,parameters=&characterEncoding=UTF-8,printConnectionTime=true,reportCSV=false,sleep=1000"	` 

## Output
All tests have a VERBOSE and SUMMARY mode. You can enable/disable them as you like.
All tests have a CSV option, when used you will get output in easy to import format for analysis

# Bugs 
* Please report any bug a
* Please suggest any test you would like to add



