package net.tc.testsuite;

import java.text.DecimalFormat;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

import net.tc.data.db.ConnectionProvider;
import net.tc.utils.Utility;

public class TestBase {
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
	DecimalFormat df2 = new DecimalFormat("#,###,###,##0.00");
	DecimalFormat dfCSV = new DecimalFormat("#.00");
	boolean reportCSV = false;

	
	
	
	
	
	void executeLocalAction(TestBase test) {
		// Generate this in the local class 
		
	}


	void init (String[] args){
	
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
		
		this.setReportCSV((this.getConfig().get("reportCSV")!=null)?Boolean.parseBoolean((String)getConfig().get("reportCSV")):false);
		
	}
	
	void localInit(String[] args) {
		/*
		 * Add in your class specific settings
		 */
	}
	
	
	Map<String, Object> getArguments(String[] args) {
		if (args == null || args.length < 1) {
			this.showHelp();
		}

		Map<String, Object> config = new HashMap();
		for (String entry : args) {

			if (entry.indexOf("parameters") > -1) {
				this.generateParameters(entry.replace("parameters=", ""));
			} else {
				String keyValue[] = entry.split("=");
				config.put(keyValue[0], keyValue[1]);
			}
		}
		config.put("parameters", this.getParameters());
		return config;
	}
 
	ConnectionProvider getConnectionProvider() {
		return connectionProvider;
	}

	void setConnectionProvider(ConnectionProvider connectionProvider) {
		this.connectionProvider = connectionProvider;
	}

	Map<String, Object> getConfig() {
		return config;
	}

	void setConfig(Map config) {
		this.config = config;
	}

	Map getParameters() {
		return parameters;
	}

	void generateParameters(String params) {

		String[] paraTem = params.split("&");
		for (String entry : paraTem) {
			if (entry.indexOf("=") > 0) {
				String[] keyValue = entry.split("=");
				getParameters().put(keyValue[0], keyValue[1]);
			}

		}

	}

	void generateConfig(String configString) {

		String[] temp = configString.split("&");
		for (String entry : temp) {
			if (entry.indexOf("=") > 0) {
				String[] keyValue = entry.split("=");
				getConfig().put(keyValue[0], keyValue[1]);
			}

		}

	}

	void setParameters(Map parameters) {
		this.parameters = parameters;
	}

	 StringBuffer showHelp() {

		StringBuffer sb = new StringBuffer();
		sb.append("******************************************\n");
		sb.append("DB Parameters to use\n");
		sb.append("Parameters are COMMA separated and the whole set must be pass as string\n");
		sb.append(
				"IE java -Xms2G -Xmx3G -classpath \"./*:./lib/*\" net.tc.testsuite.MySQLConnectionTest \"loops=10,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306\" \n");
		sb.append("url [url=jdbc:mysql://127.0.0.1:3306]\n");
		sb.append("user [user=test_user]\n");
		sb.append("password [password=test_pw]\n");
		sb.append("parameters [parameters=&useSSL=false&autoReconnect=true]\n");
		sb.append("schema [schema=test]\n");
		sb.append("\n*****************************************\nApplication Parameters \n");
		sb.append("loops [loops=50\n");
		sb.append("sleep [sleep=0] value in milliseconds \n");
		sb.append("verbose [verbose=false]\n");
		sb.append("summary [summary=false]\n");

		sb.append("reportCSV [reportCSV=false]\n");

		sb.append("****************************************\n Optional ");
		sb.append("selectForceAutocommitOff [selectForceAutocommitOff=true]\n");
		return sb;
		
		//		System.out.print(sb.toString());
//		System.exit(0);

	}




	int getLoops() {
		return loops;
	}




	void setLoops(int loops) {
		this.loops = loops;
	}




	int getSleep() {
		return sleep;
	}




	void setSleep(int sleep) {
		this.sleep = sleep;
	}




	boolean isVerbose() {
		return verbose;
	}




	void setVerbose(boolean verbose) {
		this.verbose = verbose;
	}




	boolean isSummary() {
		return summary;
	}




	void setSummary(boolean summary) {
		this.summary = summary;
	}




	static String getDefaultsConnection() {
		return defaultsConnection;
	}




	static void setDefaultsConnection(String defaultsConnection) {
		TestBase.defaultsConnection = defaultsConnection;
	}




	static String getDefaultParameters() {
		return defaultParameters;
	}




	static void setDefaultParameters(String defaultParameters) {
		TestBase.defaultParameters = defaultParameters;
	}




	int getFormattingLenghtForNano() {
		return formattingLenghtForNano;
	}




	void setFormattingLenghtForNano(int formattingLenghtForNano) {
		this.formattingLenghtForNano = formattingLenghtForNano;
	}




	DecimalFormat getDf2() {
		return df2;
	}




	void setDf2(DecimalFormat df2) {
		this.df2 = df2;
	}




	DecimalFormat getDfCSV() {
		return dfCSV;
	}




	void setDfCSV(DecimalFormat dfCSV) {
		this.dfCSV = dfCSV;
	}




	boolean isReportCSV() {
		return reportCSV;
	}




	void setReportCSV(boolean reportCSV) {
		this.reportCSV = reportCSV;
	}

}
