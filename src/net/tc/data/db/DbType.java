package net.tc.data.db;

public class DbType {

    private String name;
    private String version;
    private String platform;
    private int bitVersion;
    
    
    public DbType(String name) {
		super();
		this.name = name;
	}
	/**
     * @return the name
     */
    public final String getName() {
        return name;
    }
    /**
     * @param name the name to set
     */
    public final void setName(String name) {
        this.name = name;
    }
    /**
     * @return the version
     */
    public final String getVersion() {
        return version;
    }
    /**
     * @param version the version to set
     */
    public final void setVersion(String version) {
        this.version = version;
    }
    /**
     * @return the platform
     */
    public final String getPlatform() {
        return platform;
    }
    /**
     * @param platform the platform to set
     */
    public final void setPlatform(String platform) {
        this.platform = platform;
    }
    /**
     * @return the bitVersion
     */
    public final int getBitVersion() {
        return bitVersion;
    }
    /**
     * @param bitVersion the bitVersion to set
     */
    public final void setBitVersion(int bitVersion) {
        this.bitVersion = bitVersion;
    }
    
}
