# Generate Data
Generate data is a scial test class that I have develop to build up large datasets of USERS/Address.
The main usage for me for the SOCIAL test case in **StressTool** but you can use it for whatever you like.

To have it work correctly you need first to create a schema (whatever name) and load the data exiting in the _data_test.sql_ file.
The data is the result of a simple dump so you can reload with whatever tool you like.

The final schema will have the following tables in:

```
+-------------------+
| Tables_in_social1 |
+-------------------+
| address           |
| cities            |
| continents        |
| countries         |
| population        |
| titles            |
| users             |
+-------------------+ 
```
The tables address and users will be empty and are the ones that will be populated by the GenerateData class.

To run it:
`./run.sh GenerateData "url=jdbc:mysql://192.168.4.55:3306, user=dba,password=<secret>,verbose=true,summary=true,parameters=&characterEncoding=UTF-8,reportCSV=true,sleep=1000,printStatusDone=true,schema=bobo,savechunksize=100,numberofaddresses=1000,numberofusers=2000"	`

Where:
- savechunksize: is the set size set of rows to commit
- numberofaddresses: is the number of addresses to generate
- numberofusers: is the number of users

At the moment (unfortunately) the process is single threaded, but I will improve and will use something like in StressTool to generate more variable data

Once data is loaded. You can test the effect using [**stresstool**](https://github.com/Tusamarco/stresstool) to generate traffic.

I have just build a case **social1** to emulate a messaging platform, but to see how this works please refer to **stresstool** specific instructions.

A side note: there are almost NO index in the schema YOU MUST add them reflecting the kind of query you want to generate/use. 
 