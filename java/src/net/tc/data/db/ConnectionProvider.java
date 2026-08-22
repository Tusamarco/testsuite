package net.tc.data.db;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.lang.ref.SoftReference;
import java.sql.SQLException;
import java.util.Properties;

import javax.sql.DataSource;
import java.util.Map;

import java.sql.Connection;
import java.sql.DriverManager;

public class ConnectionProvider {
	private ConnectionInformation connInfo = null;
	private DataSource dataSource = null;
	private Map configuration = null;
	private Map config = null;

	public ConnectionProvider(Map configIn) {
		config = configIn;
		connInfo = new ConnectionInformation();

		connInfo.setConnUrl((String) config.get("url"));
		connInfo.setSchema((String) config.get("schema"));
		connInfo.setUser((String) (String) config.get("user"));
		connInfo.setPassword((String) config.get("password") != null ? (String) config.get("password") : null);
		connInfo.setDbType(ConnectionInformation.MYSQL);
		connInfo.setConnParameters(
				(String) this.getMapAsParameterString((Map<String, String>) config.get("parameters")));
		connInfo.setSelectForceAutocommitOff(
				(Boolean) Boolean.parseBoolean((String) (String) config.get("selectForceAutocommitOff")));
		connInfo.isConnectionPool();

	}

	/**
	 * @return the conInfo
	 */
	public ConnectionInformation getConnInfo() {
		return connInfo;
	}

	/**
	 * @param conInfo the conInfo to set
	 */
	public void setConnInfo(ConnectionInformation connInfo) {
		this.connInfo = connInfo;
	}

	// TODO implement support for connection pool
	public void createConnectionPool() {

	}

	public Connection getTesteMySQLConnection() throws SQLException {
		Connection conn = null;
		if (this.connInfo != null) {
			try {
				TCDataSource myDataSource = new TCDataSource();
				myDataSource.setType(ConnectionInformation.NONE);

				String connectionString = connInfo.getConnUrl() + "/" + connInfo.getSchema() + "?user="
						+ connInfo.getUser()
//     			        	    +"&useSSL=false"
						+ connInfo.getConnParameters();

				connectionString = connInfo.getPassword() != null
						? connectionString + "&password=" + connInfo.getPassword()
						: connectionString + "";

				myDataSource.setUrl(connectionString);
				this.setDataSource(myDataSource);

				conn = this.getDataSource().getConnection();
				return conn;
			} catch (Exception ex) {
				ex.printStackTrace();
			}
		}

		return null;

	}

	public Connection getSimpleMySQLConnection() throws SQLException {
		if (this.connInfo != null) {
			SoftReference<Connection> sf = null;

			if (this.getDataSource() != null) {
				try {
					sf = new SoftReference<Connection>((Connection) this.getDataSource().getConnection());

				} catch (Throwable th) {
					th.printStackTrace();

					this.setDataSource(null);
					return this.getSimpleMySQLConnection();

				}
//			if(connInfo.isSelectForceAutocommitOff()){
//				((Connection) sf.get()).setAutoCommit(false);
//			}

			} else {

				try {
					TCDataSource myDataSource = new TCDataSource();
					myDataSource.setType(ConnectionInformation.NONE);

					String connectionString = connInfo.getConnUrl() + "/" + connInfo.getSchema() + "?user="
							+ connInfo.getUser()
// 			        	    +"&useSSL=false"
							+ connInfo.getConnParameters();

					connectionString = connInfo.getPassword() != null
							? connectionString + "&password=" + connInfo.getPassword()
							: connectionString + "";

					myDataSource.setUrl(connectionString);
					this.setDataSource(myDataSource);

					sf = new SoftReference<Connection>((Connection) this.getDataSource().getConnection());
				} catch (Exception ex) {
					ex.printStackTrace();
				} finally {
					if (sf.get() != null && connInfo.isSelectForceAutocommitOff()) {
						((Connection) sf.get()).setAutoCommit(false);
					}
				}
			}
			return (Connection) sf.get();
		}

		return null;

	}

	public boolean returnConnection(Connection conn) {
		try {
			if (conn != null && !conn.isClosed()) {
				if (!conn.getAutoCommit())
					conn.commit();
				conn.close();
				conn = null;
			}
		} catch (SQLException ex) {
			ex.printStackTrace();
		}

		return false;
	}

	private DataSource getDataSource() {
		return dataSource;
	}

	private void setDataSource(DataSource dataSource) {
		this.dataSource = dataSource;
	}

	private String getMapAsParameterString(Map<String, String> in) {
		if (in == null || in.size() < 1)
			return null;

		StringBuffer sb = new StringBuffer();
		in.forEach((id, value) -> {
			sb.append("&" + id + "=" + value);
		});

		return sb.toString();
	}

}
