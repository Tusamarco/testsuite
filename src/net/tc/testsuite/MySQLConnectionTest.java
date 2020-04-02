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

public class MySQLConnectionTest extends TestBase{

	boolean printConnectionTime = true;
	ArrayList startConnTimes = new ArrayList();
	ArrayList endConnTimes = new ArrayList();
	int formattingLenghtForNano = 15;

	public static void main(String[] args) {
		MySQLConnectionTest test = new MySQLConnectionTest();
		
		if (args.length < 0 
				|| args.length > 1
				|| (args.length >= 1 && args[0].indexOf("help") > -1)) {
			
			System.out.println(test.showHelp());
			System.exit(0);
		}

		
		String[] argsLoc = args[0].replaceAll(" ","").split(",");
		
		
		test.generateConfig(defaultsConnection);
		
		test.init(argsLoc);
		test.localInit(argsLoc);
		
		test.setConnectionProvider(new ConnectionProvider(test.getConfig()));

		test.executeLocal();
		
		System.exit(0);

		
		
	}
	
	
	void localInit(String[] argsLoc) {
		
		this.setPrintConnectionTime((this.getConfig().get("printConnectionTime")!=null)?Boolean.parseBoolean((String)getConfig().get("printConnectionTime")):true);
		
	}
	
	private void executeLocal() {
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
				startOpenConenction = System.nanoTime();
				conn = this.getConnectionProvider().getTesteMySQLConnection();
				endOpenConenction = System.nanoTime() ;
				
				hostName = this.executeSQL(conn);
				
				if(getSleep() > 0 ) {
					try {
						Thread.sleep(getSleep());
					} catch (InterruptedException e) {
						// TODO Auto-generated catch block
						e.printStackTrace();
					}
				}

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
			averageReport.append("Average Time Open ns = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(avgOpen)));
			averageReport.append(" microS = " + Utility.formatNumberToPrint(8, df2.format((avgOpen / 1000))) + " \n");
			averageReport.append("Average Time Close ns = " + Utility.formatNumberToPrint(this.formattingLenghtForNano, df2.format(avgClose)));
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


	public MySQLConnectionTest() {
		this.setConfig(new HashMap());

	}

	

	 StringBuffer showHelp() {

		StringBuffer sb = super.showHelp();
		sb.append("\n****************************************\n Optional For the test");
		sb.append(" printConnectionTime [printConnectionTime=true]\n");
		

//		System.out.print(sb.toString());
//		System.exit(0);
		return sb;

	}


	private boolean isPrintConnectionTime() {
		return printConnectionTime;
	}

	private void setPrintConnectionTime(boolean printConnectionTime) {
		this.printConnectionTime = printConnectionTime;
	}



}
