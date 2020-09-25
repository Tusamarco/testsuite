package net.tc.testsuite;

import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.text.DateFormat;
import java.text.DecimalFormat;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.CountDownLatch;

import net.tc.data.db.ConnectionProvider;
import net.tc.utils.Utility;


public class QueueProcessor extends TestBase {
	int runningJobs = 1;
	int items = 1000;
	queueTable table = null;
	private CountDownLatch latch = null;
	ArrayList<Job> Jobs = new ArrayList<Job>();
	ArrayList<Job> JobsWriter = new ArrayList<Job>();
	ArrayList<Job> JobsReporting = new ArrayList<Job>();
	boolean useSkipLocked = true;
	boolean useChunks = false;
	Map<String,ArrayList<Object>> results = new HashMap<String, ArrayList<Object>>();
	int itemWriter = 0;
	int writertime = 20 ;
	static boolean runningSemaphore = false; 
	int latstRunningJob = 1 ;
	
	public static void main(String[] args) {
		QueueProcessor test = new QueueProcessor();

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
		test.setConnectionProvider(new ConnectionProvider(test.getConfig()));
		test.localInit(argsLoc);
		
		test.executequeueProcessig();

	}

	private void executequeueProcessig() {
		/*
		 * define the latch for parallel start 
		 */
		latch = new CountDownLatch(3);
		boolean running = true;
		int chunkSize = 0;
		int chunkSizeW = 0;
		//chunks calculate the size
		chunkSize = this.getItems() / this.getRunningJobs();
		if(this.getItemWriter()>0)
			chunkSizeW = this.getItems() / this.getItemWriter();
		//Create queue jobs
		QueueProcessor.setRunningSemaphore(true);
		createJobs(chunkSize);
		
		createWriters(chunkSizeW);		
    	
		this.setStartTime(System.currentTimeMillis());
    	
		startQueueJobs();
	    
	    starWriters();
	    
	    latchCountdown();
	    
	     int timeCountdown = this.getWritertime();
	     while(running) {
	    	 
	    	 if(this.getItemWriter() > 0 
	    			 && timeCountdown > 0) {
	    		 timeCountdown--;
	    		 running = true ;
	    		 boolean reloadJobs = true ;
		    	 for(int i = 0 ; i < this.getJobsWriter().size(); i++) {
			    		if(((Job)this.getJobsWriter().get(i)).getEndTime() == 0){
			    			reloadJobs = false;
			    			break;
			    		} 
			     }
		    	 if (reloadJobs) {
		    		 this.getJobsWriter().clear();
		    		 createWriters(chunkSizeW);	
		    		 starWriters();		    		 
		    		 this.latchCountdown();
		    	 }
	    		 
	    	 }
	    	 else if(this.getItemWriter() > 0 
	    			 && timeCountdown <= 0) {
	    		 
	    		 //time is over let us close the writing threads
	    		 QueueProcessor.setRunningSemaphore(false);
	    		 System.out.println("DONE");
	    		 for (Job job : this.getJobsWriter()) {
	    			 Thread ths = new Thread((Runnable) job);
	    			 try {ths.join();} catch (InterruptedException e) {e.printStackTrace();}
	    		 }
	    		 running = false;
//	    		 break;
	    	 }
	    	 else {
	    		 running = false;
	    	 }
	    	 for(int i = 0 ; i < this.getJobs().size(); i++) {
		    		if(((Job)this.getJobs().get(i)).getEndTime() == 0){
		    			running = true;
		    			break;
		    		} 
		     }
	    	 
	    	 try {Thread.sleep(2000);} catch (InterruptedException e) {e.printStackTrace();}
	     }
	    
	    this.setEndTime(System.currentTimeMillis());
	    
	    System.out.println("\n FINISH ");
	    
	    if(this.isSummary())
			this.printReport();
		
		System.exit(0);
	}

	private void starWriters() {
		//start the writers if present
	    for(Job job : this.getJobsWriter()) {
	    	Thread ths = new Thread((Runnable) job);
		    ths.start();
		    
	    }
	}

	private void createWriters(int chunkSize) {
		//create insert job if itemWriter is active
		for(int i = 1 ; i <= this.getItemWriter(); i++) {
			try {
				Job newJob = new Job();
				newJob.setId(i);
				newJob.setLatch(latch);
				newJob.setTableNameFull(this.getSchemaName()+ "." + this.getTable().getTableName());
				newJob.setMyConn(this.connectionProvider.getSimpleMySQLConnection());
				//newJob.setSkipLocked(this.isUseSkipLocked());
				newJob.setItemWriter(true);
				if(this.isUseChunks()) {
					newJob.setChunkSize(0);
//					if(i == 1) {
//					newJob.setMinItemId(1);
//					}else {
						newJob.setMinItemId((chunkSize * i) - chunkSize);
//					}
					newJob.setMaxItemId((chunkSize * i));
//					System.out.print("\n min  " + newJob.getMinItemId() + " max " + newJob.getMaxItemId());
				}
				newJob.setChunkSize(chunkSize);
				
				
				this.getJobsWriter().add(newJob);
			} catch (SQLException e) {
				e.printStackTrace();
			}
			
		}
	}

	private void latchCountdown() {
		for(int ic = 0; ic < 3 ; ic++){
	 	   try {Thread.sleep(100);
	 	   	 	 latch.countDown();
	 	   	 
	 	   }
	 	   catch (Exception e) {
	 		   e.printStackTrace();
	 	   	}
	 	}
	}

	private void startQueueJobs() {
		//we are ready to start processing the queue 
	    for(Job job : this.getJobs()) {
	    	Thread ths = new Thread((Runnable) job);
		    ths.start();
		    
	    }
	}

	private void createJobs(int chunkSize) {
		//create queue jobs
		for(int i = latstRunningJob ; i < this.getRunningJobs() + latstRunningJob; i++) {
			try {
				Job newJob = new Job();
				newJob.setId(i);
				newJob.setLatch(latch);
				newJob.setTableNameFull(this.getSchemaName()+ "." + this.getTable().getTableName());
				newJob.setMyConn(this.connectionProvider.getSimpleMySQLConnection());
				newJob.setSkipLocked(this.isUseSkipLocked());
				
				if(this.getItemWriter() > 0) {
					newJob.setLooped(true);
					newJob.setTotalItemsBunch(this.getItems());
				}
				if(this.isUseChunks()) {
					newJob.setChunkSize(0);
					if(i == latstRunningJob) {
					newJob.setMinItemId((chunkSize * i) - chunkSize);
					}else {
						newJob.setMinItemId(((chunkSize * i) - chunkSize)+ 1);
					}
					newJob.setMaxItemId((chunkSize * i));
					System.out.print("\n min  " + newJob.getMinItemId() + " max " + newJob.getMaxItemId());
				}
				newJob.setChunkSize(chunkSize);
				
				
				this.getJobs().add(newJob);
			} catch (SQLException e) {
				e.printStackTrace();
			}
			
		}
		this.latstRunningJob = (latstRunningJob + this.getRunningJobs());
	}

	private void printReport() {
		StringBuffer sb = new StringBuffer();
		Date start = new Date(((Double)this.getStartTime()).longValue());
		Date end = new Date(((Double)this.getEndTime()).longValue());
	    DateFormat df = new SimpleDateFormat("yyyy/MM/dd:HH:mm:ss");
		String pattern = "######.###";
		DecimalFormat decimalFormat = new DecimalFormat(pattern);
	    		   
		if(isReportCSV()) {

			sb.append("start time,end time,time taken total ms,"
					+ " number of jobs,number of items,avg avg total job time "
					+ "\n");
			sb.append(df.format(start) + ",");
			sb.append(df.format(end) + ",");
			sb.append((this.getEndTime() - this.getStartTime()) + ",");
			sb.append(this.getJobs().size() + ",");
			sb.append(this.getItems() + ",");
			
			long avgJobTime = 0 ;
			for(int i = 0 ; i < this.getJobs().size() ; i++) {
				avgJobTime = avgJobTime 
								+ (((Job)this.getJobs().get(i)).getEndTime() 
								- ((Job)this.getJobs().get(i)).getStartTime()) ;
			}
			avgJobTime = (avgJobTime / this.getJobs().size());
			sb.append(avgJobTime);
			
		}
		else {

			//for each job
			long avfgRecordsAvgTotal = 0;
			int totExecutedLoops = 0;
			StringBuffer sb2= new StringBuffer();
			for(int i = 0 ; i < this.getJobs().size() ; i++) {
				sb2.append("\njob id = " + (((Job)this.getJobs().get(i)).getId()));
				sb2.append("\tExecuted loops = " + (((Job)this.getJobs().get(i)).getTotLoopsExecuted()));
				sb2.append("\tprocessed items = " + (((Job)this.getJobs().get(i)).getTotProcessedItems()));
				sb2.append("\ttime taken(ms) = " + decimalFormat.format(((((Job)this.getJobs().get(i)).getExecutionTime()))/1000000));
				sb2.append("\tavg records processing time (ns) = " +((Job)this.getJobs().get(i)).getAvgItemsProcessedTime());
				avfgRecordsAvgTotal = avfgRecordsAvgTotal + ((Job)this.getJobs().get(i)).getAvgItemsProcessedTime();
				totExecutedLoops = totExecutedLoops + ((Job)this.getJobs().get(i)).getTotLoopsExecuted();
				
			}

			avfgRecordsAvgTotal = avfgRecordsAvgTotal / this.getJobs().size();
			
			sb.append("\nqueue Processor report:\n=========================\n");
			sb.append("\nStart Time                = " + start);
			sb.append("\nEnd Time                  = " + end);
			sb.append("\nTotal Time taken(ms)      = " + Utility.formatNumberToPrint(10,(this.getEndTime() - this.getStartTime())));
			sb.append("\nNumber of jobs            = " + Utility.formatNumberToPrint(10,this.getJobs().size() ));
			sb.append("\nNumber of Items processed = " + Utility.formatNumberToPrint(10,this.getItems() ));
			sb.append("\nTotal Loops executed (sum)= " + Utility.formatNumberToPrint(10,totExecutedLoops));
			sb.append("\nAvg records proc time     = " + Utility.formatNumberToPrint(10,avfgRecordsAvgTotal ));
			sb.append("\n===== JOBs section ==========");
			sb.append(sb2.toString());	
		}

		
		//sb.append(Utility.formatNumberToPrint(14, df2.format(writeTimei)) + " write Time (ns) ");
		System.out.print(sb.toString());
	}

	void localInit(String[] args) {
		if(this.getConfig().containsKey("itemwriters")) {
			this.setItemWriter(Integer.parseInt((String) this.getConfig().get("itemwriters")));
			if(this.getConfig().containsKey("writertime")) {
				this.setWritertime(Integer.parseInt((String) this.getConfig().get("writertime")));
			}
			
		}
		
		
		if(this.getConfig().containsKey("threads")) {
			this.setRunningJobs(Integer.parseInt((String) this.getConfig().get("threads")));
		}
		if(this.getConfig().containsKey("items")) {
			this.setItems(Integer.parseInt((String) this.getConfig().get("items")));
		}

		if(this.getConfig().containsKey("usechunks")) {
			this.setUseChunks(Boolean.parseBoolean((String) this.getConfig().get("usechunks")));
//			this.setUseSkipLocked(false);
		}
		
		if(this.getConfig().containsKey("skiplocked")) {
			this.setUseSkipLocked(Boolean.parseBoolean((String) this.getConfig().get("skiplocked")));
			if(this.isUseSkipLocked())
					this.setUseChunks(false);
		}
		

		this.setTable(new queueTable());
		this.getTable().setSchemaName((String) this.getConfig().get("schema"));
		this.getTable().setTableName("jobs");
		this.setConnectionProvider(new ConnectionProvider(this.getConfig()));
		try {
			if(!this.getTable().createTable(this.getConnectionProvider().getSimpleMySQLConnection())) {
				System.out.print("ERROR Cannot create the table \n" + this.getTable().getSchemaName()
						+ "." + this.getTable().getTableName());
				System.exit(1);
			};
		} catch (SQLException e) {
			e.printStackTrace();
		}
		
		if(!this.fillTable()) {
			System.out.print("ERROR Cannot fill the table \n" + this.getTable().getSchemaName()
					+ "." + this.getTable().getTableName());
			System.exit(1);
		}
		
		
	}
	
    private boolean fillTable() {
    	Connection conn =null;
    	try {
			conn = this.getConnectionProvider().getSimpleMySQLConnection();
			String sql = "Insert into " 
					+ this.getTable().getSchemaName()
					+ "." + this.getTable().getTableName()
					+ " (time_in," 
					+ "info)"
					+ "values (?,?)";
			PreparedStatement item = conn.prepareStatement(sql);
			int iCommit = 1000;
			System.out.print("\nLoading " + this.getItems() + " Items in the queue \n");
			for (int i = 0 ; i < this.getItems(); i++) {
				item.setLong(1, System.nanoTime());
				item.setString(2,"NOT PROCESSED");
				item.addBatch();
				iCommit--;
				if(iCommit ==0) {
					item.executeBatch();
					item.clearBatch();
					System.out.print(".");
//					conn.commit();
					iCommit = 1000;
					
				}
			}
			item.executeBatch();
			item.clearBatch();
//			conn.commit();
			conn.close();
			conn = null;
			return true;
		} catch (SQLException e) {
			try {
				conn.rollback();
				conn.close();
			} catch (SQLException e1) {e1.printStackTrace();}
			
			conn = null;
			e.printStackTrace();
			return false;
		}finally {
			System.out.print("\n");
		}
    }
	public int getRunningJobs() {
		return runningJobs;
	}

	public void setRunningJobs(int runningJobs) {
		this.runningJobs = runningJobs;
	}

	public int getItems() {
		return items;
	}

	public void setItems(int items) {
		this.items = items;
	}

	public queueTable getTable() {
		return table;
	}

	public void setTable(queueTable table) {
		this.table = table;
	}

	ArrayList<Job> getJobs() {
		return Jobs;
	}

	void setJobs(ArrayList<Job> jobs) {
		Jobs = jobs;
	}

	boolean isUseSkipLocked() {
		return useSkipLocked;
	}

	void setUseSkipLocked(boolean useSkipLocked) {
		this.useSkipLocked = useSkipLocked;
	}
	
	StringBuffer showHelp() {

		StringBuffer sb = super.showHelp();
		sb.append("\n****************************************\n Optional For the test: queue processor ");
		sb.append("threads [10] number of threads processing the queuque\n"
				+ "skiplocked [true] if using the SKIP LOCKED option in the SQL syntax \n"
				+ "usechunks [false] if true will divide the queue in chunks to be processed by the threads.\n It automatically disable skiplocked  \n"
				+ "items [1000] number of items to process \n"
				+ "itemwriters [0] if > then 0 the tool will also insert data while processing the time given by: \n"
				+ "\t writertime [600 (10 minutes)] time in second you want to have the writer threads to write."
				+ "");
		sb.append("=============");

//		System.out.print(sb.toString());
//		System.exit(0);
		return sb;

	}

	Map<String, ArrayList<Object>> getResults() {
		return results;
	}

	void setResults(Map<String, ArrayList<Object>> results) {
		this.results = results;
	}

	private boolean isUseChunks() {
		return useChunks;
	}

	private void setUseChunks(boolean useChunks) {
		this.useChunks = useChunks;
	}

	CountDownLatch getLatch() {
		return latch;
	}

	void setLatch(CountDownLatch latch) {
		this.latch = latch;
	}

	int getItemWriter() {
		return itemWriter;
	}

	void setItemWriter(int itemWriter) {
		this.itemWriter = itemWriter;
	}

	int getWritertime() {
		return writertime;
	}

	void setWritertime(int writertime) {
		this.writertime = writertime;
	}

	ArrayList<Job> getJobsWriter() {
		return JobsWriter;
	}

	void setJobsWriter(ArrayList<Job> jobsWriter) {
		JobsWriter = jobsWriter;
	}

	ArrayList<Job> getJobsReporting() {
		return JobsReporting;
	}

	void setJobsReporting(ArrayList<Job> jobsReporting) {
		JobsReporting = jobsReporting;
	}

	static boolean isRunningSemaphore() {
		return runningSemaphore;
	}

	static void setRunningSemaphore(boolean runningSemaphore) {
		QueueProcessor.runningSemaphore = runningSemaphore;
	}

}
class queueTable {
	String schemaName = null;
	String tableName = null; 
	int jobid = 0;
	long time_in = 0;
	long time_out = 0;
	long worked_time = 0;
	int processer = 0;
	String info = null;
	public int getJobid() {
		return jobid;
	}
	public void setJobid(int jobid) {
		this.jobid = jobid;
	}
	public long getTime_in() {
		return time_in;
	}
	public void setTime_in(long time_in) {
		this.time_in = time_in;
	}
	public long getTime_out() {
		return time_out;
	}
	public void setTime_out(long time_out) {
		this.time_out = time_out;
	}
	public long getWorked_time() {
		return worked_time;
	}
	public void setWorked_time(long worked_time) {
		this.worked_time = worked_time;
	}
	public int getProcesser() {
		return processer;
	}
	public void setProcesser(int processer) {
		this.processer = processer;
	}
	public String getInfo() {
		return info;
	}
	public void setInfo(String info) {
		this.info = info;
	}
	
	public boolean createTable(Connection conn) {
		try {
			if(conn != null
			&& !conn.isClosed()) {
				Statement stmt = conn.createStatement();
				stmt.execute("DROP TABLE IF EXISTS " + this.getSchemaName() + "." + this.getTableName());
				stmt.execute(this.getTableDefinition().toString());
				return true;
			}
		} catch (SQLException e) {
			System.out.print(this.getTableDefinition().toString());
			e.printStackTrace();
			return false;
		}
		
		
		return false;
	}
	private final StringBuffer getTableDefinition() {
		StringBuffer sb = new StringBuffer();
		sb.append("CREATE TABLE IF NOT EXISTS " + this.getSchemaName() + "." + this.getTableName());
		sb.append("(jobid int unsigned auto_increment primary key, " + 
				"time_in bigint not null, " + 
				"time_out bigint  default 0, " + 
				"worked_time bigint default 0, " + 
				"processer int unsigned default 0, " + 
				"info varchar(255),  " +
				"UPDATE_TS TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"  +
				"INDEX `idx_time_in` (time_in,time_out),  " + 
				"INDEX `idx_time_out` (time_out,time_in), " + 
				"INDEX `processer` (processer)" + 
				") engine=innodb ");
		return sb;
	}
	String getSchemaName() {
		return schemaName;
	}
	void setSchemaName(String schemaName) {
		this.schemaName = schemaName;
	}
	String getTableName() {
		return tableName;
	}
	void setTableName(String tableName) {
		this.tableName = tableName;
	}
}
class Item {
	private int id = 0 ;
	private long processtime =0;
	int getId() {
		return id;
	}
	void setId(int id) {
		this.id = id;
	}
	long getProcesstime() {
		return processtime;
	}
	void setProcesstime(long processtime) {
		this.processtime = processtime;
	}
	
}
class Job implements Runnable {
	private CountDownLatch latch = null;
	private int id = 0;
	private long startTime = 0;
	private long endTime =0;
	private long executionTime =0;
	private ArrayList<Item> processedJobs = new ArrayList<Item>();
	private Connection myConn = null;
	private String tableNameFull = null;
	private boolean skipLocked = true ; 
	private int minItemId = 0 ;
	private int maxItemId = 0 ;
	private int chunkSize = 0;
	private boolean itemWriter= false;
	private boolean isLooped = false;
    private int TotalItemsBunch = 0;
    private int totLoopsExecuted = 0;
    private int totProcessedItems = 0;
    private long avgItemsProcessedTime = 0;
    
    int iLoop = 1;
	@Override
	public void run() {
		this.setStartTime(System.currentTimeMillis());
		
		
		if(!isItemWriter()) {
			System.out.print("\nSTARTED Job ID " + this.getId() + " ItemWriter " + isItemWriter() + "");
			long execStart = System.nanoTime();
			if(this.isLooped) {
				while(QueueProcessor.isRunningSemaphore()) {
					if(iLoop > 1 && !this.isSkipLocked()) {
						this.setMinItemId(this.getMinItemId() + this.getTotalItemsBunch());
						this.setMaxItemId(this.getMaxItemId() + this.getTotalItemsBunch());
					}
					int processedItems = this.processItemList();
					this.setAvgItemsProcessedTime();
//					System.out.println(" LOOP " + this.getId() + " - " + iLoop);
					this.setTotProcessedItems((this.getTotProcessedItems() + processedItems));
					iLoop++; 
					if(this.isSkipLocked()) {
						try {Thread.sleep(100);} catch (InterruptedException e) {e.printStackTrace();}
					}
					else {
						while(!checkForNewQueue()) {
							try {Thread.sleep(500);} catch (InterruptedException e) {e.printStackTrace();}
							if(!QueueProcessor.isRunningSemaphore())
								break;
//							System.out.println(" Check failing " + this.getId() + " - " + iLoop);
						}
					}
					
				}
//				System.out.println(" LOOP " + this.getId() + " - " + iLoop);
			}
			else {
				int processedItems = this.processItemList();
				this.setTotProcessedItems(processedItems);
				this.setAvgItemsProcessedTime();
			}
			this.setTotLoopsExecuted(iLoop);
			long execEnd = System.nanoTime();
			this.setExecutionTime(this.getExecutionTime()+(execEnd - execStart));
			System.out.print("\nCLOSED Job ID " + this.getId() + " Processed items: " + getTotProcessedItems());
			

		}
		else {
			this.fillTable();
		}
		try{this.getMyConn().close();}catch(SQLException ex){ex.printStackTrace();}
		
		this.setEndTime(System.nanoTime());
//		System.out.print(" - Execution time: "+ (this.getEndTime() - this.getStartTime()));
	}
	/*
	 * This methods checks for the NEXTset of ids to process if present returns true otherwise false
	 */
	private boolean checkForNewQueue() {
		try {
			if(!this.getMyConn().isClosed()) {
				this.getMyConn().commit();
				Statement stmt = this.getMyConn().createStatement();
				String sql ="Select jobId from jobs where jobId = " + (this.getMaxItemId()+ this.getTotalItemsBunch());
				ResultSet rs = null;
				try {
					rs = stmt.executeQuery(sql);
				}
				catch (SQLException ex1) {ex1.printStackTrace(); return false;}
				if(rs.next()== false) {   
					return false;
				}
				
			}
		}
		catch(SQLException ex) {ex.printStackTrace(); return false;}
		
		return true;
	}

	private int processItemList() {
		boolean stillHaveItem = true;
		try {
			this.getMyConn().setAutoCommit(false);
			int iCount = 100;
			while(stillHaveItem == true) {
				if(!this.getMyConn().isClosed()) {
					Statement stmt = this.getMyConn().createStatement();
					Statement stmtUpdate = this.getMyConn().createStatement();
					stmt.execute("START TRANSACTION");
					String sql = "" ;
					
					if(this.getMaxItemId() == 0 && this.getChunkSize() > 0) {
						sql = "Select * from " + this.getTableNameFull() 
									+ " WHERE time_out = 0 order by jobid asc "
									+ " limit " + this.getChunkSize()  + " FOR UPDATE" ;
						if(this.isSkipLocked()) {
							sql = sql + " SKIP LOCKED" ;  
						}
					}
					else {
						sql = "Select * from " + this.getTableNameFull() 
						+ " WHERE time_out = 0  and jobid >= " 
								+ this.getMinItemId() 
								+ " and jobid < " 
								+ this.getMaxItemId() 
								+ " order by jobid asc "
						+ "  FOR UPDATE" ;
						
					}
					
					ResultSet rs = null;
					
					try {
						rs = stmt.executeQuery(sql);
					}
					catch (SQLException ex) {
						try{Thread.sleep(500);}catch (Exception ex1) {ex1.printStackTrace();}
						rs = stmt.executeQuery(sql);
					}
					if(rs.next()== false) {
						stillHaveItem = false;
					}
					else {
						rs.last();
//						System.out.println("ID " + this.getId() +" FOUND " + rs.getRow());
						rs.beforeFirst();
					}
					while(rs.next()) {
						long proctimeStart = System.nanoTime();
						Item newItem = new Item();
						int id = rs.getInt("jobid");
//						long time_in = rs.getLong("time_in");
//						String info = rs.getString("info");
//						Random r = new Random();
//						int TimeToSleep = r.nextInt((4 - 1) + 1) + 1;
//						try {Thread.sleep(100 * TimeToSleep);} catch (InterruptedException e) {e.printStackTrace();}						
						sql = "UPDATE " + this.getTableNameFull() 
						      + " SET time_out = " +  System.nanoTime()
						      + ", worked_time = " + (System.nanoTime() - proctimeStart)
						      + ", processer = " + this.getId()
						      + ", info =  'Job " + this.getId() + " was here'"
						      + " WHERE jobid = " + id;
//						rs.updateLong("time_out", proctimeEnd);
//						rs.updateLong("worked_time", (proctimeEnd - proctimeStart));
//						rs.updateInt("processer", this.getId());
//						rs.updateString("info", "Job " + this.getId() + " was here");
//						rs.updateRow();
						try {
							stmtUpdate.execute(sql);
						}
						catch (SQLException ex) {
							try{Thread.sleep(500);}catch (Exception ex1) {ex1.printStackTrace();}
							stmt.execute(sql);
						}

						
						newItem.setId(id);
						newItem.setProcesstime((System.nanoTime() - proctimeStart));
						this.getProcessedJobs().add(newItem);
						iCount-- ;
						if(iCount < 1) {
//							System.out.print(".");
							iCount = 100;
						}
								
					}//while(rs.next());
					
					stmtUpdate.execute("COMMIT");
//					long proctimeEnd = System.nanoTime();
				}
				
			}
			this.getMyConn().commit();
			
			
		}
		catch(SQLException ex) {
			ex.printStackTrace();
		}
		return this.getProcessedJobs().size();
	}
	
    private boolean fillTable() {
    	Connection conn = this.getMyConn();
    	try {
//			conn = this.getConnectionProvider().getSimpleMySQLConnection();
			String sql = "Insert into " 
					+ this.getTableNameFull()
					+ " (time_in," 
					+ "info)"
					+ "values (?,?)";
			PreparedStatement item = conn.prepareStatement(sql);
			int iCommit = 1000;
//			System.out.print("\nLoading " + this.getChunkSize() + " Items in the queue");
			for (int i = 0 ; i < this.getChunkSize(); i++) {
				item.setLong(1, System.nanoTime());
				item.setString(2,"NOT PROCESSED");
				item.addBatch();
				iCommit--;
				if(iCommit ==0) {
					item.executeBatch();
					item.clearBatch();
//					System.out.print(".");
//					conn.commit();
					iCommit = 1000;
					
				}
			}

			item.executeBatch();
			item.clearBatch();
//			conn.commit();
//			conn.close();
			conn = null;
			return true;
		} catch (SQLException e) {
			try {
				conn.rollback();
				conn.close();
			} catch (SQLException e1) {e1.printStackTrace();}
			
			conn = null;
			e.printStackTrace();
			return false;
		}
//    	finally {
//			System.out.print("\n");
//		}
    }
	CountDownLatch getLatch() {
		return latch;
	}

	void setLatch(CountDownLatch latch) {
		this.latch = latch;
	}

	int getId() {
		return id;
	}

	void setId(int id) {
		this.id = id;
	}

	long getStartTime() {
		return startTime;
	}

	void setStartTime(long startTime) {
		this.startTime = startTime;
	}

	long getEndTime() {
		return endTime;
	}

	void setEndTime(long endTime) {
		this.endTime = endTime;
	}

	ArrayList<Item> getProcessedJobs() {
		return processedJobs;
	}

	void setProcessedJobs(ArrayList<Item> processedJobs) {
		this.processedJobs = processedJobs;
	}

	Connection getMyConn() {
		return myConn;
	}

	void setMyConn(Connection myConn) {
		this.myConn = myConn;
	}

	String getTableNameFull() {
		return tableNameFull;
	}

	void setTableNameFull(String tableNameFull) {
		this.tableNameFull = tableNameFull;
	}

	boolean isSkipLocked() {
		return skipLocked;
	}

	void setSkipLocked(boolean skipLocked) {
		this.skipLocked = skipLocked;
	}

	int getMinItemId() {
		return minItemId;
	}

	void setMinItemId(int minItemId) {
		this.minItemId = minItemId;
	}

	int getMaxItemId() {
		return maxItemId;
	}

	void setMaxItemId(int maxItemId) {
		this.maxItemId = maxItemId;
	}

	int getChunkSize() {
		return chunkSize;
	}

	void setChunkSize(int chunkSize) {
		this.chunkSize = chunkSize;
	}

	boolean isItemWriter() {
		return itemWriter;
	}

	void setItemWriter(boolean itemWriter) {
		this.itemWriter = itemWriter;
	}

	boolean isLooped() {
		return isLooped;
	}

	void setLooped(boolean isLooped) {
		this.isLooped = isLooped;
	}

	int getTotalItemsBunch() {
		return TotalItemsBunch;
	}

	void setTotalItemsBunch(int totalItemsBunch) {
		TotalItemsBunch = totalItemsBunch;
	}

	int getTotLoopsExecuted() {
		return totLoopsExecuted;
	}

	void setTotLoopsExecuted(int totLoopsExecuted) {
		this.totLoopsExecuted = totLoopsExecuted;
	}

	int getTotProcessedItems() {
		return totProcessedItems;
	}

	void setTotProcessedItems(int totProcessedItems) {
		this.totProcessedItems = totProcessedItems;
	}

	long getAvgItemsProcessedTime() {
		return avgItemsProcessedTime;
	}

	void setAvgItemsProcessedTime() {
		long locAvg = 0 ;
		long locTot = 0;
		for(Item item:this.getProcessedJobs()) {
			locTot = locTot + item.getProcesstime();
		}
		locAvg = locTot / this.getProcessedJobs().size();
				
		if(iLoop > 1) {
			this.avgItemsProcessedTime = (avgItemsProcessedTime + locAvg)/2; // /this.iLoop;
		}
		else {
			this.avgItemsProcessedTime = (avgItemsProcessedTime + locAvg);
		}
	}

	long getExecutionTime() {
		return executionTime;
	}

	void setExecutionTime(long executionTime) {
		this.executionTime = executionTime;
	}

	void setAvgItemsProcessedTime(long avgItemsProcessedTime) {
		this.avgItemsProcessedTime = avgItemsProcessedTime;
	}

}
