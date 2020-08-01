package net.tc.testsuite;

import java.sql.Array;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.text.DecimalFormat;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

import net.tc.data.db.ConnectionProvider;
import net.tc.utils.MathU;
import net.tc.utils.Utility;

public class StaleReadTest extends TestBase {

	ConnectionProvider connectionProviderRead = null;
	private int rowsNumber = 10000;
	long toleranceNanosec = 5000 ; //Tolerance by default is 5 microsecond if more than that we have an issue
	Map<String,ArrayList<Long>> results = new HashMap<String, ArrayList<Long>>();
	int staleReads = 0; 
	
	public static void main(String[] args) {
		StaleReadTest test = new StaleReadTest();

		if (args.length <= 0 
				|| args.length > 1
				|| (args.length >= 1 && args[0].indexOf("help") > -1)
				|| args[0].equals("")) {
			
			System.out.println(test.showHelp());
			System.exit(0);
		}

		String[] argsLoc = args[0].replaceAll(" ","").split(",");
		
		
		test.generateConfig(defaultsConnection);
		
		test.init(argsLoc);
		test.localInit(argsLoc);
		
		test.setConnectionProvider(new ConnectionProvider(test.getConfig()));
		if (test.getSleep()<=0)test.setSleep(2000);
		
		if(test.getConfig().containsKey("rowsNumber")){
			test.setRowsNumber(Integer.parseInt((String) test.getConfig().get("rowsNumber")));
		}

	
		if(test.getConfig().containsKey("urlRead")){
			Map<String,Object> newConfig = test.getConfig();
			newConfig.put("url", test.getConfig().get("urlRead"));
			test.setConnectionProviderRead(new ConnectionProvider(newConfig));
		}
		else
		{
			test.setConnectionProviderRead(new ConnectionProvider(test.getConfig()));
		}
			
		
		test.executeLocal();
		
		System.exit(0);

		
	}

	private void executeLocal() {
		// TODO create procedure to simulate stale reads
		
		/**
		 * The main idea here is to connect to two different servers or more in PXC or Innodb cluster
		 * And test if there is any discrepancy between write and reads
		 * 
		 * To do so:
		 * 1) create a table large enough to have a some meaningful dimension
		 * 
		 */
		Connection writeConn = null;
		Connection readConn = null;
		try {
			writeConn = this.getConnectionProvider().getSimpleMySQLConnection();
			readConn  = this.getConnectionProviderRead().getSimpleMySQLConnection();
		} catch (SQLException e) {
			e.printStackTrace();
		}
		
		if(!this.createTable(writeConn))
			System.out.print(" Error Cannot create table ");

		if(!this.fillTable(writeConn))
			System.out.print(" Error Cannot fill table with data");
		
		if(!this.executeStaleReadTest(writeConn,readConn))
			System.out.print(" Error Cannot execute test");
		
		this.getConnectionProvider().returnConnection(writeConn);
		this.getConnectionProviderRead().returnConnection(readConn);
		
		if(this.isSummary())
			this.printReport();
		
		System.exit(0);
				
	}

	private void printReport() {
		
		

		StringBuffer averageReport = new StringBuffer();
		
		double averageWrite = MathU.getAverage(this.getResults().get("writeTime"));
		double averageRead = MathU.getAverage(this.getResults().get("readTime"));
		double averageLag = MathU.getAverage(this.getResults().get("lagTime"));
		double stdWrite = MathU.calculateStandardDeviation(this.getResults().get("writeTime"), averageWrite);
		double stdRead = MathU.calculateStandardDeviation(this.getResults().get("readTime"), averageRead);
		double stdLag = MathU.calculateStandardDeviation(this.getResults().get("lagTime"), averageLag);
		
		long[] maxminWrite = MathU.getMaxMin(this.getResults().get("writeTime"));
		long[] maxminRead = MathU.getMaxMin(this.getResults().get("readTime"));
		long[] maxminLag = MathU.getMaxMin(this.getResults().get("lagTime"));
		
		int istancesWithLag = this.getStaleReads();  
		float pct  = (float) 0.0;
		if(this.getLoops() > this.getRowsNumber()) {		
			  pct = ((istancesWithLag *100)/this.getRowsNumber());
		}
		else {
			pct = ((istancesWithLag *100)/this.getLoops());
		}
		
		if(isReportCSV()) {
			String pattern = "######.###";
			DecimalFormat decimalFormat = new DecimalFormat(pattern);

			averageReport.append("loops,slate reads found,stale read % on total,"
					+ " total time,avg write time,avg read time,avg lag time,"
					+ " min write,max write,min read,max read,max lag, min lag, "
					+ "\n");

			averageReport.append( this.getRowsNumber());
			averageReport.append("," + istancesWithLag );
			averageReport.append("," + pct );
			averageReport.append("," + decimalFormat.format(this.getExecutionTime()));
			averageReport.append("," + decimalFormat.format(averageWrite));
			averageReport.append("," + decimalFormat.format(averageRead));
			averageReport.append("," + decimalFormat.format(averageLag));
			
			averageReport.append("," + decimalFormat.format(maxminWrite[0]) + "," + decimalFormat.format(maxminWrite[1] ));
			averageReport.append("," + decimalFormat.format(maxminRead[0]) + "," + decimalFormat.format(maxminRead[1] ));
			averageReport.append("," + decimalFormat.format(maxminLag[0]) + "," + decimalFormat.format(maxminLag[1] ));
			
			averageReport.append("," + decimalFormat.format(stdWrite ));
			averageReport.append("," + decimalFormat.format(stdRead ));
			averageReport.append("," + decimalFormat.format(stdRead ));
			
			
		}
		else {
			String pattern = "###,###.###";
			DecimalFormat decimalFormat = new DecimalFormat(pattern);
			averageReport.append("\n============ Summary ===========");
			averageReport.append("\nTotal loops  = " + this.getLoops());
			averageReport.append("\nTotal rows  = " + this.getRowsNumber());
			averageReport.append("\nStale reads found = " + istancesWithLag );
			averageReport.append("\nStale reads found% = " + pct );
			averageReport.append("\n============ Time in nano seconds");
			averageReport.append("\nTotal execution time = " + decimalFormat.format(this.getExecutionTime()));
			averageReport.append("\nAverage write time = " + decimalFormat.format(averageWrite));
			averageReport.append("\nAverage read time = " + decimalFormat.format(averageRead));
			averageReport.append("\nAverage lag time = " + decimalFormat.format(averageLag));
			
			averageReport.append("\nMax/Min Write time = " + decimalFormat.format(maxminWrite[0]) + "/" +decimalFormat.format( maxminWrite[1] ));
			averageReport.append("\nMax/Min Read time = " + decimalFormat.format(maxminRead[0]) + "/" + decimalFormat.format(maxminRead[1] ));
			averageReport.append("\nMax/Min lag time = " + decimalFormat.format(maxminLag[0]) + "/" + decimalFormat.format(maxminLag[1] ));
			
			averageReport.append("\nstd Dev Write time = " + decimalFormat.format(stdWrite ));
			averageReport.append("\nstd Dev Read time = " + decimalFormat.format(stdRead ));
			averageReport.append("\nstd Dev lag time = " + decimalFormat.format(stdRead ));
			
			
		}

		System.out.println(averageReport.toString());
		
	}

	private boolean executeStaleReadTest(Connection writeConn, Connection readConn) {
		try {
			StringBuffer sb = new StringBuffer();
			Statement wstmt = writeConn.createStatement();
			Statement rstmt = readConn.createStatement();
			System.out.println("Going to sleep for two seconds before executing");
			Thread.sleep(this.getSleep());
			System.out.println("Executing:");
						
			this.setStartTime(System.nanoTime());
			
			ArrayList<Integer> ids = getIds(wstmt);
			
			ArrayList<Long> writeTime = new ArrayList<Long>();
			ArrayList<Long> readTime = new ArrayList<Long>();
			ArrayList<Long> lagTime = new ArrayList<Long>();
			
			String dateRun = Utility.getTimeStamp(System.currentTimeMillis()) ;  
			
			int iCounter = 0;
			
			if(!this.isReportCSV()) {
				sb.append(Utility.formatStringToPrint(6,"ID") +" (" + Utility.formatNumberToPrint(6," #loop") +" Exceeding the tollerance of "+ this.getToleranceNanosec()+")\n" );
			}
			else {
				sb.append("ID,#loop,writeTime,readTime,lagTime\n");
			}
			
			for(Object id:ids.toArray()) {
				iCounter++;
				long writeTimei = 0;
				long readTimei = 0;
				long lagTimei = 0;
				boolean lag = false;
						
				String sqlW ="";
				if(Utility.isEvenNumber(iCounter)) {
					sqlW = "UPDATE " + this.getSchemaName() + ".staleread SET t = '"+ dateRun +"' WHERE id = " + id.toString();
				}
				else {
					sqlW = "REPLACE INTO " + this.getSchemaName() +".staleread VALUES ("+ id.toString() +", uuid(), time('" + dateRun + "') , (FLOOR( 1 + RAND( ) *60 )))";
				}
				String sqlR = "Select id  From " + this.getSchemaName() + ".staleread WHERE id = " + id.toString() + " and t = '"+ dateRun +"'"; 
				
				long startTimeWrite = System.nanoTime();
				
				wstmt.execute(sqlW);
				wstmt.execute("COMMIT");
				writeConn.commit();
				
				long startTimeRead = System.nanoTime();
				writeTimei = (startTimeRead - startTimeWrite);
				
				if(this.isReportCSV()) {				
					sb.append(id +"," + iCounter +"," );
				}else {
					sb.append(Utility.formatStringToPrint(6,id.toString()) +" (" + Utility.formatNumberToPrint(6,iCounter) +")" );
				}

				long[] checkRecords = {0,0};
				while(checkRecords[0] < 1) {
					checkRecords = checkRecord(rstmt, sqlR);
					if(checkRecords[0] <1)
						lag = true;
					readTimei += checkRecords[1]; 
				}
				long endTimeRead= System.nanoTime();

				if(lag)
					this.setStaleReads(this.getStaleReads()+1);
				
				lagTimei = (endTimeRead - startTimeRead);
				
				
				
				writeTime.add(writeTimei);
				readTime.add(readTimei);
				lagTime.add(lagTimei);
				
				
				if(this.isVerbose())
						print_verbose(sb, iCounter, id, writeTimei, readTimei, lagTimei, lag);
				else
					System.out.print(".");
				
				sb.delete(0, sb.length());
				//If loop is < than rows force the exit based on loop #
				if(this.getLoops() < this.getRowsNumber()
					&& iCounter >= this.getLoops()) {
					break;
				}
			}
			System.out.println("\n Done \n ");
			this.getResults().put("writeTime", writeTime);
			this.getResults().put("readTime", readTime);
			this.getResults().put("lagTime", lagTime);
			
			this.setEndTime(System.nanoTime());
			
			return true;
		}catch(Throwable th){th.printStackTrace();}

		return false;
	}

	private void print_verbose(StringBuffer sb, int iCounter, Object id, long writeTimei, long readTimei, long lagTimei,
			boolean lag) {
		
		if(this.isReportCSV()) {
			sb.append(id + "," + iCounter + "," + writeTimei +"," + readTimei +"," + lagTimei );
			System.out.println(sb.toString());
		}
		else {
			if (lag 
				&& (lagTimei  > this.getToleranceNanosec())) {
				sb.append(Utility.formatNumberToPrint(14, df2.format(writeTimei)) + " write Time (ns) ");
				sb.append(Utility.formatNumberToPrint(14, df2.format(readTimei)) + " read Time (ns)  ");
				sb.append(Utility.formatNumberToPrint(14, df2.format(lagTimei)) + " lag Time (ns)   ");
				System.out.println(sb.toString());

			}

		}
	}

	private long[] checkRecord(Statement rstmt, String sqlR) {
		long[] values = new long[2];
		values[0] = 0;
		
		try {
				long readStart = System.nanoTime();
				
				ResultSet rs = rstmt.executeQuery(sqlR);
				if(rs !=null 
						&& rs.last()
						&& rs.getRow() > 0 ) {
					rs.close();
					values[0]=1;
				}
				values[1] = (System.nanoTime() - readStart);
		}
		catch(SQLException ex) {
			ex.printStackTrace();
		}
		
		return values;
	}
	
	private ArrayList<Integer> getIds(Statement wstmt) throws SQLException {
		ArrayList<Integer> ids = new ArrayList<Integer>();
		ResultSet rs = wstmt.executeQuery("SELECT id from "+ this.getSchemaName() + ".staleread");
		while(rs.next()) {
			ids.add(rs.getInt(1));
		}
		rs.close();
		wstmt.clearWarnings();
		return ids;
	}

	private boolean fillTable(Connection writeConn) {
		String dateStart = Utility.getTimeStamp(System.currentTimeMillis()) ; 
		System.out.println("Loading table ... please wait");
		
		Statement stmt = null;
		try {
			String insert = "INSERT INTO " + this.getSchemaName() +".staleread VALUES (NULL, uuid(), time('" + dateStart + "'),  (FLOOR( 1 + RAND( ) *60 )));";
			
			writeConn.setAutoCommit(false);
			stmt = writeConn.createStatement();
			
			int totRows=0;
			while(totRows <= getRowsNumber()) {
				for( int subBatch=1 ; subBatch <= 100;subBatch++) {
					stmt.addBatch(insert);
					totRows++ ; 
				}
				stmt.executeBatch();
				writeConn.commit();
				stmt.clearBatch();
				System.out.println(" wrote " + totRows + " of " + this.getRowsNumber());
				
			}
			
		} catch (SQLException e) {
			e.printStackTrace();
		}
		finally {
			if(stmt != null) {
				try {stmt.close();} catch (SQLException e) {e.printStackTrace();}
				stmt = null;
				System.out.println("Loading table ... done");
				return true;
			}
		}
		
		return false;
	}

	private boolean createTable(Connection connWrite) {
		
		StringBuffer sb = new StringBuffer();
		
//		'CREATE TABLE IF NOT EXISTS `'.$schema.'`.`joinit` (
//		  `i` bigint(11) NOT NULL AUTO_INCREMENT,
//		  `s` char(255) DEFAULT NULL,
//		  `t` datetime NOT NULL,
//		  `g` bigint(11) NOT NULL,
//		  KEY(`i`, `t`),
//		  PRIMARY KEY(`i`)
//		) ENGINE=$engine  DEFAULT CHARSET=utf8;'
		
		
		String drop = "Drop table if exists `" + this.getSchemaName() + "`.`staleread`;";
		sb.append("CREATE TABLE IF NOT EXISTS `" + this.getSchemaName()  +"`.`staleread` (");
		sb.append("`id` bigint(11) NOT NULL AUTO_INCREMENT,");
		sb.append("`s` char(255) DEFAULT NULL,");
		sb.append("`t` datetime NOT NULL,");
		sb.append("`g` bigint(11) NOT NULL,");
		sb.append(" KEY(`id`, `t`),");
		sb.append(" PRIMARY KEY(`id`)");
		sb.append(" ) ENGINE=InnoDB  DEFAULT CHARSET=utf8;");
		
		Statement stmt = null;
		try {
			stmt = connWrite.createStatement();
			stmt.execute(drop);
			stmt.execute(sb.toString());
		} catch (SQLException e) {
			e.printStackTrace();
		}
		finally {
			if(stmt != null) {
				try {stmt.close();} catch (SQLException e) {e.printStackTrace();}
				stmt = null;
				return true;
			}
		}
		return false;
		
	}
	
	 StringBuffer showHelp() {

		StringBuffer sb = super.showHelp();
		sb.append("\n****************************************\n Optional For the test: Stale read ");
		sb.append(" urlRead=jdbc:mysql://127.0.0.1:3307 \n"
				+ " if present the tool will compare the WRITES done against url\n"
				+ " with the reads from urlRead "
				+ " I not present the tool assume the presence of ProxySQL and will use only one URL"
				+ "\n"
				+ "rowsNumber [rowsNumber=10000]\n"
				+ "");
		sb.append("=============");
		sb.append("sleep in this context refer to the time the test will wait after table load\n"
				+ "Default sleep = 2000 ms (2 seconds)");
		
		

//		System.out.print(sb.toString());
//		System.exit(0);
		return sb;

	}

	private ConnectionProvider getConnectionProviderRead() {
		return connectionProviderRead;
	}

	private void setConnectionProviderRead(ConnectionProvider connectionProviderRead) {
		this.connectionProviderRead = connectionProviderRead;
	}

	private int getRowsNumber() {
		return rowsNumber;
	}

	private void setRowsNumber(int rowsNumber) {
		this.rowsNumber = rowsNumber;
	}

	private long getToleranceNanosec() {
		return toleranceNanosec;
	}

	private void setToleranceNanosec(long toleranceNanosec) {
		this.toleranceNanosec = toleranceNanosec;
	}

	private Map<String, ArrayList<Long>> getResults() {
		return results;
	}

	private void setResults(Map<String, ArrayList<Long>> results) {
		this.results = results;
	}

	private int getStaleReads() {
		return staleReads;
	}

	private void setStaleReads(int staleReads) {
		this.staleReads = staleReads;
	}
	
	
}
