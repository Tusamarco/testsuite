package net.tc.testsuite;

import net.tc.data.db.ConnectionProvider;

public class StaleReadTest extends TestBase {

	
	
	
	
	public static void main(String[] args) {
		StaleReadTest test = new StaleReadTest();

		if (args.length == 0 
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

	private void executeLocal() {
		// TODO Auto-generated method stub
		
	}

	
	
	
	
	
}
