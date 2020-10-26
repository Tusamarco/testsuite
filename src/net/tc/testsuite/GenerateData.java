package net.tc.testsuite;

import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.text.DateFormat;
import java.text.DecimalFormat;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.Random;
import java.util.Set;
import java.util.TreeMap;

import org.apache.commons.math3.distribution.AbstractRealDistribution;
import org.apache.commons.math3.distribution.NormalDistribution;
import org.apache.commons.math3.distribution.ParetoDistribution;
import org.apache.commons.math3.distribution.UniformRealDistribution;
import org.apache.commons.math3.distribution.WeibullDistribution;

import net.tc.data.db.ConnectionProvider;
import net.tc.utils.Utility;

public class GenerateData extends TestBase {
	static final int RANDOM_UNIFORM= 1;
	static final int RANDOM_GAUSSIAN= 2;
	static final int RANDOM_PARETO= 3;
	
	ArrayList name = new ArrayList();
	ArrayList lastName = new ArrayList();
	int numberOfAddresses = 100;
	int numberOfUsers = 100;
	int randomType = GenerateData.RANDOM_UNIFORM;
	AbstractRealDistribution randomGenerator = null;
	int saveChunkSize = 500;
	long userExecutionTime = 0 ;
	long addressExecutionTime =0;
	
	
	public static void main(String[] args) {
		GenerateData dataCreator = new GenerateData();

		if (args.length <= 0 
				|| args.length > 1
				|| (args.length >= 1 && args[0].indexOf("help") > -1)
				|| args[0].equals("")) {
			
			System.out.println(dataCreator.showHelp());
			System.exit(0);
		}

		String[] argsLoc = args[0].replaceAll(" ","").split(",");
		
		
		dataCreator.generateConfig(defaultsConnection);
		
		dataCreator.init(argsLoc);
		dataCreator.setConnectionProvider(new ConnectionProvider(dataCreator.getConfig()));
		dataCreator.localInit(argsLoc);
		
		dataCreator.executeLoad();

	}

	void executeLoad() {
		this.setStartTime(System.currentTimeMillis());
		AddressBuilder addressBuilder = new AddressBuilder();
		UserBuilder userBuilder = new UserBuilder();
		try {
			addressBuilder.setSchemaName(this.getSchemaName());
			addressBuilder.setVerboseReport(this.isVerbose());
			addressBuilder.setSaveCunkSize(this.getSaveChunkSize());
			addressBuilder.setRandomGenerator(this.getRandomGenerator());
			this.addressExecutionTime = System.currentTimeMillis();
			addressBuilder.loadData( this.getConnectionProvider().getSimpleMySQLConnection());
			addressBuilder.ComposeAddress(this.getNumberOfAddresses());
			this.addressExecutionTime = (System.currentTimeMillis() - this.addressExecutionTime);

			userBuilder.setSchemaName(this.getSchemaName());
			userBuilder.setVerboseReport(this.isVerbose());
			userBuilder.setSaveCunkSize(this.getSaveChunkSize());
			userBuilder.setRandomGenerator(this.getRandomGenerator());
			this.userExecutionTime = System.currentTimeMillis();;
			userBuilder.loadData(this.getConnectionProvider().getSimpleMySQLConnection());
			userBuilder.ComposeUsers(numberOfUsers);
			this.userExecutionTime = (System.currentTimeMillis() - this.userExecutionTime);
			
			
		}catch (SQLException ex) {
			ex.printStackTrace();
		}
		this.setEndTime(System.currentTimeMillis());
	    if(this.isSummary())
			this.printReport();
		
		System.exit(0);
	}
	void localInit(String[] args) {
		if(this.getConfig().containsKey("numberofaddresses")) {
			this.setNumberOfAddresses(Integer.parseInt((String) this.getConfig().get("numberofaddresses")));
		}
		if(this.getConfig().containsKey("numberofusers")) {
				this.setNumberOfUsers(Integer.parseInt((String) this.getConfig().get("numberofusers")));
		}
		if(this.getConfig().containsKey("savechunksize")) {
			this.setSaveChunkSize(Integer.parseInt((String) this.getConfig().get("savechunksize")));
  	    }

		if(this.getConfig().containsKey("randomtype")) {
			String randomTypeConfig = (String) this.getConfig().get("randomtype");
			switch (randomTypeConfig.toLowerCase()) {
			case "uniform": this.setRandomType(GenerateData.RANDOM_UNIFORM);this.setRandomGenerator(Utility.getUniformRandomGenerator(1));break;
			case "gaussian": this.setRandomType(GenerateData.RANDOM_GAUSSIAN);this.setRandomGenerator(Utility.getGaussianRandomGenerator(500));break;
			case "pareto": this.setRandomType(GenerateData.RANDOM_PARETO);this.setRandomGenerator(Utility.getParetoRandomGenerator(25));break;
			}
		}
		else {
			this.setRandomType(GenerateData.RANDOM_UNIFORM);
			this.setRandomGenerator(Utility.getUniformRandomGenerator(1));
			System.out.println("[WARNING] No definition for random generato. Default [UNIFORM] is used");
		}
		
	}

	private void printReport() {
		Date start = new Date(((Double)this.getStartTime()).longValue());
		Date end = new Date(((Double)this.getEndTime()).longValue());
		StringBuffer sb = new StringBuffer();
		
		sb.append("=================================\n");
		sb.append("Data generation report\n");
		sb.append("Data generation Started : " + start + " \n");
		sb.append("Data generation Ended : " + end + " \n");
		sb.append("Time taken (sec) : " + (this.getEndTime() - this.getStartTime())/1000 + " \n");
		sb.append("Users generated: " + this.getNumberOfUsers()+ " Time taken (sec) " + (this.userExecutionTime)/1000 + " \n");
		sb.append("Address generated: " + this.getNumberOfAddresses()+ " Time taken (sec) " + this.addressExecutionTime/1000 + " \n");
		sb.append("=================================\n");
		
		
		System.out.print(sb.toString());
	}
	
	
	StringBuffer showHelp() {

		StringBuffer sb = super.showHelp();
		sb.append("\n****************************************\n Optional For the test: Generate data \n");
		sb.append("numberofaddresses = Number of address to generate [100]\n");
		sb.append("numberofusers = Number of users to generate [100]\n");
		sb.append("savechunksize = The chuck dimension to flush to tble in batch [100]\n");
		sb.append("randomtype = [uniform] possible types {uniform|gaussian|pareto} \n\t some values are generated using custom random algorythm");
		sb.append("\n=============\n");
		sb.append("Example of parameters 'url=jdbc:mysql://192.168.4.55:3306, \n"
				+ "\t\t user=dba,password=<secret>,\n"
				+ "\t\t verbose=true,\n"
				+ "\t\t summary=true,\n"
				+ "\t\t parameters=&characterEncoding=UTF-8,\n"
				+ "\t\t reportCSV=true,\n"
				+ "\t\t printStatusDone=true,\n"
				+ "\t\t savechunksize=100,\n"
				+ "\t\t numberofaddresses=10000,\n"
				+ "\t\t numberofusers=20000\n");
		
		

//		System.out.print(sb.toString());
//		System.exit(0);
		return sb;

	}

	ArrayList getName() {
		return name;
	}

	void setName(ArrayList name) {
		this.name = name;
	}

	ArrayList getLastName() {
		return lastName;
	}

	void setLastName(ArrayList lastName) {
		this.lastName = lastName;
	}

	int getNumberOfAddresses() {
		return numberOfAddresses;
	}

	void setNumberOfAddresses(int numberOfAddresses) {
		this.numberOfAddresses = numberOfAddresses;
	}

	int getNumberOfUsers() {
		return numberOfUsers;
	}

	void setNumberOfUsers(int numberOfUsers) {
		this.numberOfUsers = numberOfUsers;
	}

	int getRandomType() {
		return randomType;
	}

	void setRandomType(int randomType) {
		this.randomType = randomType;
	}

	AbstractRealDistribution getRandomGenerator() {
		return randomGenerator;
	}

	void setRandomGenerator(AbstractRealDistribution randomGenerator) {
		this.randomGenerator = randomGenerator;
	}

	int getSaveChunkSize() {
		return saveChunkSize;
	}

	void setSaveChunkSize(int saveChunkSize) {
		this.saveChunkSize = saveChunkSize;
	}	
}
class AddressBuilder{
	TreeMap<Integer, City> cities = new TreeMap();
	TreeMap<String, Country> countries = new TreeMap();
	TreeMap <Integer,String>streets = new TreeMap();
	AbstractRealDistribution randomGenerator = null;
	int saveCunkSize = 0;
	String schemaName = null;
	boolean verboseReport =false; 
	
	
	AddressBuilder(){}
	
	AddressBuilder (Connection connIn){
		this.conn = connIn;
	}

	Connection conn = null;

	TreeMap <String,Address>addresses = new TreeMap();
	
	Address getAddress(String key){
		if(addresses != null
				&& addresses.size() >0) {
			return addresses.get(key);
		}
		
		return null;
	}
	void loadData(Connection conn){
		if(conn != null) {
			this.conn = conn;
		}
		else {
			System.out.println("[ERROR] - connection is null");
		}
			

		this.loadCities();
//		this.getCountry();
		this.loadStreets();
		

  }
  int saveCunck(ArrayList<Address> toSaveAddresses) {
	  int[] saved = {0} ;
	  if(toSaveAddresses != null & toSaveAddresses.size() > 0) {
		  
		  try {
				  this.conn.setAutoCommit(false);
				  if(!this.conn.isClosed()) {
						Statement stmt = this.conn.createStatement();
						Statement stmtUpdate = this.conn.createStatement();
						String insertHeader = "INSERT INTO "+ this.getSchemaName() +".address (cityId,city,country_iso3,region,street,active,link_id) " ;
						stmt.execute("START TRANSACTION");
//						ResultSet rs = null;
						
						StringBuffer sb = new StringBuffer();
						for(Address address :toSaveAddresses) {
							sb.delete(0, sb.length());
							sb.append("Values(");
							sb.append(address.getCityId() + ",");
							sb.append("'" + address.getCityName().replace("'", "") + "',");
							sb.append("'" + address.getIso3() + "',"); 
							sb.append("'" + address.getSubRegion().replace("'","") + "',");
							sb.append("'" + address.getStreet().replace("'", "") + "',");
							sb.append(address.isActive() + ",");
							sb.append(address.getLinkid() );
							sb.append(")");
							stmt.addBatch(insertHeader + sb.toString());
//							System.out.println(insertHeader + sb.toString());
						}
						try{
								saved = stmt.executeBatch();
						}
						catch(SQLException eex) {
							stmt.execute("rollback");
							stmt.close();
							eex.printStackTrace();
						}
						stmt.execute("COMMIT");
						stmt.close();
						return saved[0];
				  }
				  else {
					  System.out.println("[ERROR] - cannot save chunk conn is null");
					  
				  }
			  
		  }
		  catch(SQLException ex) {
			  ex.printStackTrace();
		  }
		  
		  
	  }
	  
	  return 0;
  }
  long ComposeAddress(int numberOfinstances ){
	  
		long min = 1;
		long max = this.getCities().size() -1;
		ArrayList<Address> toSaveAddresses = new ArrayList();
        int addressSaveCountdown = this.getSaveCunkSize();
        int numberOfinstancesCountDown = numberOfinstances;
        
        long start = System.currentTimeMillis();
		
		for(int i = 0; i <numberOfinstances; i++) {
			if(this.getCities() == null) {
				System.out.println("[ERROR] Cities are not loaded");
				System.exit(2);
			}
			Object[] mycityKeys =  this.getCities().keySet().toArray();
			Long myIndex = this.getLongFromRandomGenerator(min, max, 0);
			Object myK = mycityKeys[myIndex.intValue()];
			City myCity = (City)this.getCities().get(myK);
			
			Address myAddress = new Address();
			myAddress.setCityId(myCity.getId());
			myAddress.setCityName(myCity.getName());
			myAddress.setSubRegion(myCity.getSubRegion());
			myAddress.setIso3(myCity.getIso3());
			myAddress.setActive(Utility.isEvenNumber(myIndex.intValue()));
			myAddress.setLinkid(myCity.getId());

			Object[] myStreetsKeys =  this.getStreets().keySet().toArray();
			myIndex = this.getLongFromRandomGenerator(min, myStreetsKeys.length -1, 0);
			myK = myStreetsKeys[myIndex.intValue()];
			myAddress.setStreet(this.getStreets().get(myK));
			
			if(addressSaveCountdown > 1) {
				toSaveAddresses.add(myAddress);
				addressSaveCountdown--;
			}
			else {
				toSaveAddresses.add(myAddress);
				if(this.saveCunck(toSaveAddresses) > 0) {
					numberOfinstancesCountDown = numberOfinstancesCountDown - this.getSaveCunkSize();
					if(this.isVerboseReport()) {
					System.out.println("[INFO] Saved " + this.getSaveCunkSize() 
							+ " Addresses to go " + numberOfinstancesCountDown +"/" + numberOfinstances);
					}
					toSaveAddresses.clear();
					addressSaveCountdown = this.getSaveCunkSize();
				}
				
			}
//			double dd= Utility.getNumberFromParetoRandomMinMax(min, max,25);
//			double dd2= Utility.getNumberFromUniformRandomMinMax(min, max);
//			double dd3= Utility.getNumberFromGaussianRandomMinMax(min, max,500);
//			double dd4= Utility.getNumberFromRandomMinMax(min, max);
//			System.out.println(Math.round(dd) + "\t" +Math.round(dd2) + "\t" + Math.round(dd3) +"\t" + Math.round(dd4) );
		}
		
	  return System.currentTimeMillis() - start;
  }
	
  TreeMap<Integer, City> loadCities() {
		try {
			long start = System.currentTimeMillis();
			System.out.println("Starting retreve cities ");
			this.conn.setAutoCommit(false);
			if(!this.conn.isClosed()) {
				Statement stmt = this.conn.createStatement();
				Statement stmtUpdate = this.conn.createStatement();
				stmt.execute("START TRANSACTION");
				ResultSet rs = null;
				rs = stmt.executeQuery("select id,name,country_iso3,subcountry from "+ this.getSchemaName() +".cities order by id" );
				while(rs.next()) {
					Integer id = rs.getInt("id");
					String name = rs.getString("name");
					String subRegion = rs.getString("subcountry");
					String iso3 = rs.getString("country_iso3");
					City city = new City(id,name,iso3,subRegion);
					this.cities.put(city.getId(), city);
				}
				rs.close();
				stmt.execute("COMMIT");
				stmt.close();
			}
			long end = System.currentTimeMillis();
			System.out.println("Loaded " + this.cities.size() + " cities. End retreve cities time taken (ms) = " + (end-start));
		
		
		} catch (SQLException e) {
		
			e.printStackTrace();
		}
		return null;
	}
   TreeMap getCountry() {
		try {
			long start = System.currentTimeMillis();
			System.out.println("Starting retreve countries ");
			this.conn.setAutoCommit(false);
			if(!this.conn.isClosed()) {
				Statement stmt = this.conn.createStatement();
				Statement stmtUpdate = this.conn.createStatement();
				stmt.execute("START TRANSACTION");
				ResultSet rs = null;
				rs = stmt.executeQuery("select iso3,official_name_en from "+ this.getSchemaName() +".countries order by iso3" );
				while(rs.next()) {
					String iso3 = rs.getString("iso3");
					String name = rs.getString("official_name_en");
					Country country = new Country(iso3,name);
					this.countries.put(country.getIso3(), country);
				}
				rs.close();
				stmt.execute("COMMIT");
				stmt.close();
			}
			long end = System.currentTimeMillis();
			System.out.println("Loaded " + this.countries.size() + " countries. End retreve countries time taken (ms) = " + (end-start));
		
		
		} catch (SQLException e) {
		
			e.printStackTrace();
		}
		return null;
	}

   TreeMap loadStreets() {
		try {
			long start = System.currentTimeMillis();
			System.out.println("Starting retreve streets ");
			this.conn.setAutoCommit(false);
			if(!this.conn.isClosed()) {
				Statement stmt = this.conn.createStatement();
				Statement stmtUpdate = this.conn.createStatement();
				stmt.execute("START TRANSACTION");
				ResultSet rs = null;
				rs = stmt.executeQuery("select id,streetname from "+ this.getSchemaName() +".streets order by id" );
				while(rs.next()) {
					Integer id = rs.getInt("id");
					String name = rs.getString("streetname");					
					this.streets.put(id, name);
				}
				rs.close();
				stmt.execute("COMMIT");
				stmt.close();
			}
			long end = System.currentTimeMillis();
			System.out.println("Loaded " + this.streets.size() + " streets. End retreve streets time taken (ms) = " + (end-start));
		
		
		} catch (SQLException e) {
		
			e.printStackTrace();
		}
		return null;
	}

	AbstractRealDistribution getRandomGenerator() {
		return randomGenerator;
	}
	
	void setRandomGenerator(AbstractRealDistribution randomGenerator) {
		this.randomGenerator = randomGenerator;
	}
	long  getLongFromRandomGenerator(long min, long max, long range) {
	  	if(min >= max) return new Long(max);
	  	
		long maxL = 0;
		if(this.getRandomGenerator() instanceof NormalDistribution) {
			maxL = Math.round(this.getRandomGenerator().sample() * range  +(max/2));
		}
		else {
			maxL = (long) Math.round(Math.round(max * this.getRandomGenerator().sample()));
		}
		maxL = maxL> max?getLongFromRandomGenerator(min,max,range):maxL;
		maxL=maxL<min?getLongFromRandomGenerator(min,max,range):maxL; 
	  	return maxL;
		
		
	}

	TreeMap<Integer, City> getCities() {
		return cities;
	}

	void setCities(TreeMap<Integer, City> cities) {
		this.cities = cities;
	}

	TreeMap<Integer, String> getStreets() {
		return streets;
	}

	void setStreets(TreeMap<Integer, String> streets) {
		this.streets = streets;
	}

	int getSaveCunkSize() {
		return saveCunkSize;
	}

	void setSaveCunkSize(int saveCunkSize) {
		this.saveCunkSize = saveCunkSize;
	}

	String getSchemaName() {
		return schemaName;
	}

	void setSchemaName(String schemaName) {
		this.schemaName = schemaName;
	}

	boolean isVerboseReport() {
		return verboseReport;
	}

	void setVerboseReport(boolean verboseReport) {
		this.verboseReport = verboseReport;
	} 
  
  
}
class UserBuilder{
	TreeMap <Integer,User> users = new TreeMap();
    ArrayList<String> name = new ArrayList();
    ArrayList<String> lastName = new ArrayList();
    String schemaName = null;
    boolean verboseReport =false;
    
	Connection conn = null;
	AbstractRealDistribution randomGenerator = null;
	int saveCunkSize = 0;
	
	User getUser(Integer id) {
		if(users != null
		   && users.size() > 0 ) 
		{
			return users.get(id);
		}
		return null;
	}
	TreeMap<Integer, User> getUsers() {
		return users;
	}
	void setUsers(TreeMap<Integer, User> users) {
		this.users = users;
	}
	void loadData(Connection conn){
		if(conn != null) {
			this.conn = conn;
		}
		else {
			System.out.println("[ERROR] - connection is null");
		}
			

		this.loadLastName();
		this.loadName();
	}	
		
	void loadName() {
			try {
				long start = System.currentTimeMillis();
				System.out.println("Starting retreve names  ");
				this.conn.setAutoCommit(false);
				if(!this.conn.isClosed()) {
					Statement stmt = this.conn.createStatement();
					Statement stmtUpdate = this.conn.createStatement();
					stmt.execute("START TRANSACTION");
					ResultSet rs = null;
					rs = stmt.executeQuery("select name from "+ this.getSchemaName() +".firstname" );
					while(rs.next()) {
						String name = rs.getString("name");
						this.getName().add(name);
					}
					rs.close();
					stmt.execute("COMMIT");
					stmt.close();
				}
				long end = System.currentTimeMillis();
				System.out.println("Loaded " + this.getName().size() + " names. End retreve names time taken (ms) = " + (end-start));
			
			
			} catch (SQLException e) {
			
				e.printStackTrace();
			}
	}

	void loadLastName() {
		try {
			long start = System.currentTimeMillis();
			System.out.println("Starting retreve lastName  ");
			this.conn.setAutoCommit(false);
			if(!this.conn.isClosed()) {
				Statement stmt = this.conn.createStatement();
				Statement stmtUpdate = this.conn.createStatement();
				stmt.execute("START TRANSACTION");
				ResultSet rs = null;
				rs = stmt.executeQuery("select name from "+ this.getSchemaName() +".lastname" );
				while(rs.next()) {
					String name = rs.getString("name");
					this.getLastName().add(name);
				}
				rs.close();
				stmt.execute("COMMIT");
				stmt.close();
			}
			long end = System.currentTimeMillis();
			System.out.println("Loaded " + this.getLastName().size() + " Lastnames. End retreve lastNames time taken (ms) = " + (end-start));
		
		
		} catch (SQLException e) {
		
			e.printStackTrace();
		}
	} 
	
	  long ComposeUsers(int numberOfinstances ){
		  
			long min = 1;
			long maxN = this.getName().size() -1;
			long maxLN = this.getLastName().size() -1;
			
			long start = System.currentTimeMillis();
			
	        int userSaveCountdown = this.getSaveCunkSize();
	        int numberOfinstancesCountDown = numberOfinstances;
	        ArrayList toSaveUsers = new ArrayList();
			long nowMs = System.currentTimeMillis();
			int maxAddressId = this.getMaxAddressId();
				        
			for(int i = 0; i <numberOfinstances; i++) {
				if(this.getName() == null) {
					System.out.println("[ERROR] Names are not loaded");
					System.exit(2);
				}
				Long myIndexN = this.getLongFromRandomGenerator(min, maxN, 0);
				Long myIndexLN = this.getLongFromRandomGenerator(min, maxLN, 0);
				int userAge = Utility.getNumberFromGaussianRandomMinMax(14, 90,15).intValue();
				long registrationDateMs = Utility.getNumberFromUniformRandomMinMax(1262304000000L,nowMs);
				String registrationDate = Utility.getTimeStamp(registrationDateMs);
				int addressId = ((Long)this.getLongFromRandomGenerator(1, maxAddressId, 0)).intValue();
				boolean active = Utility.isEvenNumber(addressId);
				String gender = Utility.isEvenNumber(registrationDateMs)?"M":"F";
				User myUser = new User();
				String name = (String) this.getName().toArray()[myIndexN.intValue()];
				String lastName = (String) this.getLastName().toArray()[myIndexLN.intValue()];
				
				myUser.setActive(active);
				myUser.setAddressId(addressId);
				myUser.setAge(userAge);
				myUser.setEmail((lastName.replace("'", "") + "." + name.replace("'", "") + "@gmail.com").toLowerCase());
				myUser.setGender(gender);
				myUser.setLastname(lastName.replace("'", ""));
				myUser.setName(name.replace("'", ""));
				myUser.setRegistrationDate(registrationDate);
				myUser.setPhone(Utility.getPhoneNumber());
				

				
				if(userSaveCountdown > 1) {
					toSaveUsers.add(myUser);
					userSaveCountdown--;
				}
				else {
					toSaveUsers.add(myUser);
					if(this.saveCunck(toSaveUsers) > 0) {
						numberOfinstancesCountDown = numberOfinstancesCountDown - this.getSaveCunkSize();
						if(this.isVerboseReport()) {
							System.out.println("[INFO] Saved " + this.getSaveCunkSize() 
									+ " Users to go " + numberOfinstancesCountDown +"/" + numberOfinstances);
						}
						toSaveUsers.clear();
						userSaveCountdown = this.getSaveCunkSize();
					}
					
				}
//				double dd= Utility.getNumberFromParetoRandomMinMax(min, max,25);
//				double dd2= Utility.getNumberFromUniformRandomMinMax(min, max);
//				double dd3= Utility.getNumberFromGaussianRandomMinMax(min, max,500);
//				double dd4= Utility.getNumberFromRandomMinMax(min, max);
//				System.out.println(Math.round(dd) + "\t" +Math.round(dd2) + "\t" + Math.round(dd3) +"\t" + Math.round(dd4) );
			}
			
		  
		  return System.currentTimeMillis() - start;
	  }	

	  private int getMaxAddressId() {
		try {
		  long start = System.currentTimeMillis();
			this.conn.setAutoCommit(false);
			Integer id = 0;
			if(!this.conn.isClosed()) {
				Statement stmt = this.conn.createStatement();
				Statement stmtUpdate = this.conn.createStatement();
				stmt.execute("START TRANSACTION");
				ResultSet rs = null;
				rs = stmt.executeQuery("select max(id) id from "+ this.getSchemaName() +".address" );
				while(rs.next()) {
					id = rs.getInt("id");
				}
				rs.close();
				stmt.execute("COMMIT");
				stmt.close();
			}
			long end = System.currentTimeMillis();
//			System.out.println("Loaded " + this.cities.size() + " cities. End retreve cities time taken (ms) = " + (end-start));
			
			return id;
		
		} catch (SQLException e) {
		
			e.printStackTrace();
		}		 
		  
		  
		return 0;
	}
	  
	/*
		+-------------------+-------------------+------+-----+---------+-------------------+
		| Field             | Type              | Null | Key | Default | Extra             |
		+-------------------+-------------------+------+-----+---------+-------------------+
		| id                | int unsigned      | NO   | PRI | NULL    | auto_increment    |
		| name              | varchar(50)       | NO   |     | NULL    |                   |
		| lastname          | varchar(50)       | NO   |     | NULL    |                   |
		| age               | smallint unsigned | NO   |     | NULL    |                   |
		| phone             | varchar(50)       | NO   |     | NULL    |                   |
		| registration_date | datetime          | NO   |     | NULL    |                   |
		| address_id        | int unsigned      | NO   |     | NULL    |                   |
		| email             | varchar(150)      | NO   |     | NULL    |                   |
		| active            | tinyint unsigned  | NO   |     | 0       | DEFAULT_GENERATED |
		| gender            | char(1)           | YES  |     | NULL    |                   |
		+-------------------+-------------------+------+-----+---------+-------------------+
	*/  
	  
	int saveCunck(ArrayList<User> toSaveUsers) {
		  int[] saved = {0} ;
		  if(toSaveUsers != null & toSaveUsers.size() > 0) {
			  
			  try {
					  this.conn.setAutoCommit(false);
					  if(!this.conn.isClosed()) {
							Statement stmt = this.conn.createStatement();
							Statement stmtUpdate = this.conn.createStatement();
							String insertHeader = "INSERT INTO "+ this.getSchemaName() +".users (name,lastname,age,address_id,phone,registration_date,email,active,gender) " ;
							stmt.execute("START TRANSACTION");
//							ResultSet rs = null;
							
							StringBuffer sb = new StringBuffer();
							for(User user :toSaveUsers) {
								sb.delete(0, sb.length());
								sb.append("Values(");
								sb.append("'" + user.getName() + "',");
								sb.append("'" + user.getLastname() + "',");
								sb.append(user.getAge() + ",");
								sb.append(user.getAddressId() + ",");
								sb.append("'" + user.getPhone() + "',");
								sb.append("'" + user.getRegistrationDate() + "',");
								sb.append("'" + user.getEmail() + "',");
								sb.append(user.isActive() + ",");
								sb.append("'" +user.getGender() + "'");
								sb.append(")");
								stmt.addBatch(insertHeader + sb.toString());
//								System.out.println(insertHeader + sb.toString());
							}
							try{
									saved = stmt.executeBatch();
							}
							catch(SQLException eex) {
								stmt.execute("rollback");
								stmt.close();
								eex.printStackTrace();
							}
							stmt.execute("COMMIT");
							stmt.close();
							return saved[0];
					  }
					  else {
						  System.out.println("[ERROR] - cannot save chunk conn is null");
						  
					  }
				  
			  }
			  catch(SQLException ex) {
				  ex.printStackTrace();
			  }
			  
			  
		  }
		  
		  return 0;
	  }	  

	  long  getLongFromRandomGenerator(long min, long max, long range) {
	  	if(min >= max) return new Long(max);
	  	
		long maxL = 0;
		if(this.getRandomGenerator() instanceof NormalDistribution) {
			maxL = Math.round(this.getRandomGenerator().sample() * range  +(max/2));
		}
		else {
			maxL = (long) Math.round(Math.round(max * this.getRandomGenerator().sample()));
		}
		maxL = maxL> max?getLongFromRandomGenerator(min,max,range):maxL;
		maxL=maxL<min?getLongFromRandomGenerator(min,max,range):maxL; 
	  	return maxL;
		
	  }

	  
	  
	Connection getConn() {
		return conn;
	}

	
	void setConn(Connection conn) {
		this.conn = conn;
	}
	ArrayList getName() {
		return name;
	}
	void setName(ArrayList name) {
		this.name = name;
	}
	ArrayList getLastName() {
		return lastName;
	}
	void setLastName(ArrayList lastName) {
		this.lastName = lastName;
	}
	AbstractRealDistribution getRandomGenerator() {
		return randomGenerator;
	}
	void setRandomGenerator(AbstractRealDistribution randomGenerator) {
		this.randomGenerator = randomGenerator;
	}
	int getSaveCunkSize() {
		return saveCunkSize;
	}
	void setSaveCunkSize(int saveCunkSize) {
		this.saveCunkSize = saveCunkSize;
	}
	String getSchemaName() {
		return schemaName;
	}
	void setSchemaName(String schemaName) {
		this.schemaName = schemaName;
	}
	boolean isVerboseReport() {
		return verboseReport;
	}
	void setVerboseReport(boolean verboseReport) {
		this.verboseReport = verboseReport;
	}	
	
}
/*
+---------+------------------+------+-----+---------+-------------------+
| Field   | Type             | Null | Key | Default | Extra             |
+---------+------------------+------+-----+---------+-------------------+
| id      | int unsigned     | NO   | PRI | NULL    | auto_increment    |
| active  | tinyint unsigned | NO   |     | 0       | DEFAULT_GENERATED |
| street  | varchar(100)     | NO   |     | NULL    |                   |
| city    | varchar(50)      | NO   |     | NULL    |                   |
| region  | varchar(50)      | NO   |     | NULL    |                   |
| country | varchar(50)      | NO   |     | NULL    |                   |
| link_id | int unsigned     | NO   |     | NULL    |                   |
+---------+------------------+------+-----+---------+-------------------+
*/

class Address{
	boolean active = true;
	String street = null;
	int cityId = 0;
	String cityName = null;
	String iso3 = null;
	String subRegion = null;
	int linkid = 0 ;
	boolean isActive() {
		return active;
	}
	 void setActive(boolean active) {
		this.active = active;
	}
	 String getStreet() {
		return street;
	}
	 void setStreet(String street) {
		this.street = street;
	}
	 int getCityId() {
		return cityId;
	}
	 void setCityId(int cityId) {
		this.cityId = cityId;
	}
	 String getIso3() {
		return iso3;
	}
	 void setIso3(String iso3) {
		this.iso3 = iso3;
	}
	 int getLinkid() {
		return linkid;
	}
	 void setLinkid(int linkid) {
		this.linkid = linkid;
	}
	String getSubRegion() {
		return subRegion;
	}
	void setSubRegion(String subRegion) {
		this.subRegion = subRegion;
	}
	void save(Connection conn){
		
		
	}
	String getCityName() {
		return cityName;
	}
	void setCityName(String cityName) {
		this.cityName = cityName;
	}
	
	
} 

class User{
	String name = null;
	String lastname= null;
	int age = 0 ;
	String phone = null;
	String registrationDate = null;
	int addressId = 0 ;
	String email = null;
	boolean active = true;
	String gender = null;
	 String getName() {
		return name;
	}
	 void setName(String name) {
		this.name = name;
	}
	 String getLastname() {
		return lastname;
	}
	 void setLastname(String lastname) {
		this.lastname = lastname;
	}
	 int getAge() {
		return age;
	}
	 void setAge(int age) {
		this.age = age;
	}
	 String getPhone() {
		return phone;
	}
	 void setPhone(String phone) {
		this.phone = phone;
	}
	 String getRegistrationDate() {
		return registrationDate;
	}
	 void setRegistrationDate(String registrationDate) {
		this.registrationDate = registrationDate;
	}
	 int getAddressId() {
		return addressId;
	}
	 void setAddressId(int addressId) {
		this.addressId = addressId;
	}
	 String getEmail() {
		return email;
	}
	 void setEmail(String email) {
		this.email = email;
	}
	 boolean isActive() {
		return active;
	}
	 void setActive(boolean active) {
		this.active = active;
	}
	 String getGender() {
		return gender;
	}
	 void setGender(String gender) {
		this.gender = gender;
	}
			
	
}
class City{
	City(){}
	City(Integer id, String name, String iso3, String subRegion){
		this.setId(id);
		this.setName(name);
		this.setIso3(iso3);
		this.setSubRegion(subRegion);
	}
	
	int id = 0;
	String name = null;
	String iso3 = null;
	String subRegion = null;
	int getId() {
		return id;
	}
	void setId(int id) {
		this.id = id;
	}
	String getName() {
		return name;
	}
	void setName(String name) {
		this.name = name;
	}
	String getIso3() {
		return iso3;
	}
	void setIso3(String iso3) {
		this.iso3 = iso3;
	}
	String getSubRegion() {
		return subRegion;
	}
	void setSubRegion(String subRegion) {
		this.subRegion = subRegion;
	}
	
}

class Country{
	Country(){}
	
	Country(String iso3,String name){
		this.setIso3(iso3);
		this.setName(name);
	}
	
	String iso3 = null;
	String name = null;
	String getIso3() {
		return iso3;
	}
	void setIso3(String iso3) {
		this.iso3 = iso3;
	}
	String getName() {
		return name;
	}
	void setName(String name) {
		this.name = name;
	}
}
