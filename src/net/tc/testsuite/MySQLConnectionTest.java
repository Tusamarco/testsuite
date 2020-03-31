package net.tc.testsuite;

import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.text.DecimalFormat;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

import net.tc.data.db.ConnectionProvider;
import net.tc.utils.Utility;

public class MySQLConnectionTest {

	ConnectionProvider connectionProvider = null;
	Map config = null;
	int loops = 50;
	int sleep = 0;
	boolean verbose = false;
	boolean summary = false;
	boolean printConnectionTime = true;
	static String defaultsConnection = "&url=jdbc:mysql://127.0.0.1:3306&user=test_user&password=test_password&schema=test";
	static String defaultParameters = "&useSSL=false&autoReconnect=true";
	Map parameters = new HashMap();
	ArrayList startConnTimes = new ArrayList();
	ArrayList endConnTimes = new ArrayList();
	int formattingLenghtForNano = 15;
	DecimalFormat df2 = new DecimalFormat( "#,###,###,##0.00" );
	DecimalFormat dfCSV = new DecimalFormat( "#.00" );
	boolean reportCSV = false;

	public static void main(String[] args) {

		if (args.length == 0 
				|| args.length > 1
				|| (args.length >= 1 && args[0].indexOf("help") > -1))
			showHelp();

		String[] argsLoc = args[0].replaceAll(" ","").split(",");
		
		MySQLConnectionTest test = new MySQLConnectionTest();
		test.generateConfig(defaultsConnection);
		
		test.init(argsLoc);
		
		test.setConnectionProvider(new ConnectionProvider(test.getConfig()));

		test.executeLoops();
		
		System.exit(0);

	}
	
	private void executeLoops() {
		long startOpenConenction = 0;
		long endOpenConenction = 0;
		long startCloseConenction = 0;		
		long endCloseConenction = 0;
		String hostName = null;
		
		try {
			Connection conn = null;
			
			if(isReportCSV()) {
				System.out.println((isVerbose()? "SqlOutput,":"")+"OpenTime(ns),OpenTime(us),CloseTime(ns),CloseTime(us)");
			}
			
			for(int iLoop = 0; iLoop <= this.getLoops(); iLoop++) {
				if(getSleep() > 0 ) {
					try {
						Thread.sleep(getSleep());
					} catch (InterruptedException e) {
						// TODO Auto-generated catch block
						e.printStackTrace();
					}
				}
				
				startOpenConenction = System.nanoTime();
				conn = this.getConnectionProvider().getTesteMySQLConnection();
				endOpenConenction = System.nanoTime() ;
				
				hostName = this.executeSQL(conn);
				
				startCloseConenction = System.nanoTime();
				this.getConnectionProvider().returnConnection(conn);
				endCloseConenction = System.nanoTime();
				
				startConnTimes.add(endOpenConenction - startOpenConenction);
				endConnTimes.add(endCloseConenction - startCloseConenction);
				
				long open = (long) startConnTimes.get((startConnTimes.size()-1));
				long close = (long) endConnTimes.get((endConnTimes.size() -1));
				long msOpen = (open / 1000);
				long msClose = (close / 1000);
				
				
				if(this.isPrintConnectionTime()) {
					if(!isReportCSV()) {
						System.out.println(
							(isVerbose()? hostName + " ":"")
							+ "Open Connection time (nano) = " 
							+ Utility.formatNumberToPrint(this.getFormattingLenghtForNano(), df2.format(open))
							+ " microS = " + Utility.formatNumberToPrint(10, df2.format(msOpen))
							+ " Close Connection time (nano) = "
							+ Utility.formatNumberToPrint(this.getFormattingLenghtForNano(), df2.format(close))
							+ " microS = " + Utility.formatNumberToPrint(10, df2.format(msClose))
							);
						}
					else {
						System.out.println(
								(isVerbose()? hostName + ",":"")
								+ open +","
								+ msOpen + ","
								+ close + ","
								+ msClose
						);		
						
					}
							
				}
				
				
			}
			
			
			
		} catch (SQLException e) {
			
			e.printStackTrace();
		}
		if(this.isSummary())
			System.out.println(calculateSummary());
	}

	private String calculateSummary() {
		long totOpen = 0;
		long totClose = 0;
		long maxOpen = 0;
		long maxClose = 0;
		long minOpen = Long.MAX_VALUE;
		long minClose = Long.MAX_VALUE;
		
		//Taking the number from instance 1 and not 0 to skip the object initialization cost
		for(int i = 1 ; i < this.startConnTimes.size(); i++){
			totOpen = (totOpen + (long)this.startConnTimes.get(i));
			totClose= (totClose + (long) this.endConnTimes.get(i));
			maxOpen = ((long)this.startConnTimes.get(i)) > maxOpen?((long)this.startConnTimes.get(i)):maxOpen;
			maxClose = ((long)this.endConnTimes.get(i)) > maxClose?((long)this.endConnTimes.get(i)):maxClose;
			minOpen = ((long)this.startConnTimes.get(i)) < minOpen?((long)this.startConnTimes.get(i)):minOpen;
			minClose = ((long)this.endConnTimes.get(i)) < minClose?((long)this.endConnTimes.get(i)):minClose;		
        }
		double avgOpen = (totOpen/(startConnTimes.size() -1));
		double avgClose = (totClose/(startConnTimes.size() -1));
		
		/*
		 * calculate standard deviation
		 */
		double standardDevOpen = 0.0;
		double standardDevClose = 0.0;
		ArrayList stdOpen = new ArrayList();
		ArrayList stdClose = new ArrayList();
		
		for(int i = 1 ; i < this.startConnTimes.size(); i++){
			
			stdOpen.add(Math.pow((long)this.startConnTimes.get(i) - avgOpen, 2));
			stdClose.add(Math.pow((long)this.endConnTimes.get(i) - avgClose, 2));
		}
		
		for(int i = 0 ; i < stdOpen.size(); i++){
			standardDevOpen  += (double)stdOpen.get(i);
			standardDevClose += (double)stdClose.get(i);
			
		}
		
		standardDevOpen = standardDevOpen/(stdOpen.size());
		standardDevClose = standardDevClose/(stdClose.size());
		
		standardDevOpen = Math.sqrt(standardDevOpen);
		standardDevClose = Math.sqrt(standardDevClose);
		
		StringBuffer averageReport = new StringBuffer();
		
		if(!isReportCSV()) {
			
			averageReport.append("\n Summary \n");
			averageReport.append("Average Time Open  = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(avgOpen)));
			averageReport.append(" microS = " + Utility.formatNumberToPrint(8, df2.format((avgOpen / 1000))) + " \n");
			averageReport.append("Average Time Close = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(avgClose)));
			averageReport.append(" microS = " + Utility.formatNumberToPrint(8, df2.format((avgClose / 1000))) + " \n");
			
			averageReport.append("Max Open (time in nano seconds)  = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(maxOpen)) + "\n");
			averageReport.append("Max Close (time in nano seconds) = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(maxClose)) + "\n");

			averageReport.append("Min Open (time in nano seconds)  = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(minOpen)) + "\n");
			averageReport.append("Min Close (time in nano seconds) = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(minClose)) + "\n");


			averageReport.append("\n Standard deviation \n");
			averageReport.append("StdDev connection Open   = " + Utility.formatNumberToPrint(15, df2.format(standardDevOpen)) + "\n");
			averageReport.append("StdDev connection Close  = " + Utility.formatNumberToPrint(15, df2.format(standardDevClose)) + "\n");
			
		}
		else {
			
			averageReport.append("AVG_Open(ns),AVG_open(us),AVG_close(ns),AVG_close(us),MaxOpen(ns),MaxClose(ns),MinOpen(ns),MinClose(ns),StdvOpen,StdvClose\n");
			averageReport.append(
					dfCSV.format(avgOpen) +","
					+ dfCSV.format((avgOpen / 1000)) +"," 
					+ dfCSV.format(avgClose) +"," 
					+ dfCSV.format((avgClose/1000)) + ","
					+ dfCSV.format(maxOpen) +","
					+ dfCSV.format(maxClose) + ","
					+ dfCSV.format(minOpen) + ","
					+ dfCSV.format(minClose) + ","
					+ dfCSV.format(standardDevOpen) + ","
					+ dfCSV.format(standardDevClose) 
					);
		}
		
		return averageReport.toString();
		
	}

	private String executeSQL(Connection conn) {
		String hostname = null;
		try {
			Statement stmt = conn.createStatement();
			ResultSet rs = stmt.executeQuery("SELECT @@hostname as name");
			
			while(rs.next()){
				hostname = rs.getString("name");
			}
			rs.close();
			stmt.close();
		}
		catch(SQLException ex) {
			ex.printStackTrace();
		}
		return hostname;
	}

	private void init(String[] args) {
		
		try {
			/*
			 * Configuration first
			 */
			this.generateParameters(defaultParameters);
			if (args.length >= 1) {
				this.getConfig().putAll(this.getArguments(args));
//				this.setConfig(this.getArguments(args));
			}
		} catch (Throwable th) {
			th.printStackTrace();
		}
		
		
		
//		setConnectionProvider(new ConnectionProvider(getConfig()));
		
		
		this.setLoops((this.getConfig().get("loops")!=null)
				?Utility.isNumeric(this.getConfig().get("loops"))?(int)getConfig().get("loop")
						:Integer.parseInt((String)getConfig().get("loops")):50); 
		
		this.setSleep((this.getConfig().get("sleep")!=null)
				?Utility.isNumeric(this.getConfig().get("sleep"))?(int)getConfig().get("sleep")
						:Integer.parseInt((String)getConfig().get("sleep")):0); 
		
		
		this.setVerbose((this.getConfig().get("verbose")!=null)?Boolean.parseBoolean((String)getConfig().get("verbose")):false);
		this.setSummary((this.getConfig().get("summary")!=null)?Boolean.parseBoolean((String)getConfig().get("summary")):false);
		this.setPrintConnectionTime((this.getConfig().get("printConnectionTime")!=null)?Boolean.parseBoolean((String)getConfig().get("printConnectionTime")):true);
		this.setReportCSV((this.getConfig().get("reportCSV")!=null)?Boolean.parseBoolean((String)getConfig().get("reportCSV")):false);
	}
	public MySQLConnectionTest() {
		this.setConfig(new HashMap());

	}

	

	private static void showHelp() {

		StringBuffer sb = new StringBuffer();
		sb.append("******************************************\n");
		sb.append("DB Parameters to use\n");
		sb.append("Parameters are COMMA separated and the whole set must be pass as string\n");
		sb.append("IE java -Xms2G -Xmx3G -classpath \"./*:./lib/*\" net.tc.testsuite.MySQLConnectionTest 'loops=10,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306' \n");
		sb.append("url [url=jdbc:mysql://127.0.0.1:3306]\n");
		sb.append("user [user=test_user]\n");
		sb.append("password [password=test_pw]\n");
		sb.append("parameters [parameters=&useSSL=false&autoReconnect=true]\n");
		sb.append("schema [schema=test]\n");
		sb.append("\n*****************************************\nApplication Parameters \n");
		sb.append("loops [loops=50\n");
		sb.append("sleep [sleep=0]\n");
		sb.append("verbose [verbose=false]\n");
		sb.append("summary [summary=false]\n");
		sb.append("printConnectionTime [printConnectionTime=true]\n");
		sb.append("reportCSV [reportCSV=false]\n");

		sb.append("\n\n****************************************\n Optional ");
		sb.append("selectForceAutocommitOff [selectForceAutocommitOff=true]\n\n");
		System.out.print(sb.toString());
		System.exit(0);

	}

	private Map<String, Object> getArguments(String[] args) {
		if (args.length ==0 ) {
			MySQLConnectionTest.showHelp();
		}

		Map<String, Object> config = new HashMap();
		for (String entry : args) {
			
			if(entry.indexOf("parameters") > -1) {
				this.generateParameters(entry.replace("parameters=", ""));
			}
			else {
				String keyValue[] = entry.split("=");
				config.put(keyValue[0], keyValue[1]);
			}
		}
		config.put("parameters", this.getParameters());
		return config;
	}

	private ConnectionProvider getConnectionProvider() {
		return connectionProvider;
	}

	private void setConnectionProvider(ConnectionProvider connectionProvider) {
		this.connectionProvider = connectionProvider;
	}

	private Map<String, Object> getConfig() {
		return config;
	}

	private void setConfig(Map config) {
		this.config = config;
	}

	private int getLoops() {
		return loops;
	}

	private void setLoops(int loops) {
		this.loops = loops;
	}

	private int getSleep() {
		return sleep;
	}

	private void setSleep(int sleep) {
		this.sleep = sleep;
	}

	private boolean isVerbose() {
		return verbose;
	}

	private void setVerbose(boolean verbose) {
		this.verbose = verbose;
	}

	private boolean isSummary() {
		return summary;
	}

	private void setSummary(boolean summary) {
		this.summary = summary;
	}

	private boolean isPrintConnectionTime() {
		return printConnectionTime;
	}

	private void setPrintConnectionTime(boolean printConnectionTime) {
		this.printConnectionTime = printConnectionTime;
	}

	private Map getParameters() {
		return parameters;
	}
    
	private void  generateParameters(String params) {
		
		String[] paraTem = params.split("&");
		for (String entry : paraTem) { 
			if(entry.indexOf("=") > 0) {
				String[] keyValue = entry.split("=");
				getParameters().put(keyValue[0], keyValue[1]);
			}
		
		}
		

	}
	private void  generateConfig(String configString) {
		
		String[] temp = configString.split("&");
		for (String entry : temp) { 
			if(entry.indexOf("=") > 0) {
				String[] keyValue = entry.split("=");
				getConfig().put(keyValue[0], keyValue[1]);
			}
		
		}
		

	}
	private void setParameters(Map parameters) {
		this.parameters = parameters;
	}

	private int getFormattingLenghtForNano() {
		return formattingLenghtForNano;
	}

	private void setFormattingLenghtForNano(int formattingLenghtForNano) {
		this.formattingLenghtForNano = formattingLenghtForNano;
	}

	private boolean isReportCSV() {
		return reportCSV;
	}

	private void setReportCSV(boolean reportCSV) {
		this.reportCSV = reportCSV;
	}

}
