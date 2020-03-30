package net.tc.utils;

import java.util.ArrayList;
import java.util.Map;

public class MathU {
/* static class to provide math functions
 * 
 */
    public void Math() {
	
    }
    /** 
     * Calculate average from a range (Array) of Long Objects
     * @param values Long[]
     * @return Double
     */
    public static Double getAverage(Long[] values){
	if(values != null && values.length !=0){
		long sum = 0;
		for(Long i : values){
		    sum += i.longValue();
		}
		if(sum > 0)
		    return new Double(sum/values.length);
	}
	
	return null;
    }
    
    /** 
     * Calculate average from a range (Array) of Long Objects
     * @param values long[]
     * @return Double
     */
    public static Double getAverage(long[] values){
	
	return null;
    }
    
    /** 
     * Calculate average from a Map (Array) of Long Objects
     * @param values long[]
     * @return Double
     */
    public static Double getAverage(Map values){
	
	return null;
    }
	public static Double getAverage(ArrayList<Long> values) {
		if(values != null && values.size() !=0){
			long sum = 0;
			for(Long i : values){
			    sum += i.longValue();
			}
			if(sum > 0)
			    return new Double(sum/values.size());
		}
		
		return 0.0;
		
	}
    
}
