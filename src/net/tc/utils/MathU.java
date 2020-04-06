package net.tc.utils;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
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
	   public static double calculateStandardDeviation(ArrayList<Long> values, double average) {
		   
			/*
			 * calculate standard deviation
			 */
			double standardDevOpen = 0.0;
			ArrayList<Double> stdOpen = new ArrayList();

			for(int i = 1 ; i < values.size(); i++){
				stdOpen.add(Math.pow((long)values.get(i) - average, 2));
			}
		
			for(int i = 0 ; i < stdOpen.size(); i++){
				standardDevOpen  += (double)stdOpen.get(i);
				
			}
			
			standardDevOpen = standardDevOpen/(stdOpen.size());
			standardDevOpen = Math.sqrt(standardDevOpen);
		   
		   return standardDevOpen;
	   }
	   
	   public static long[] getMaxMin(ArrayList<Long> values) {
		   long[] returnValue = {0,0};
		   long min = Long.MAX_VALUE;
		   long max = 0;
		   for(int i =0 ; i < values.size(); i++) {
			   min = values.get(i) < min?values.get(i):min;
			   max = values.get(i) > max?values.get(i):max; 
		   }
		   returnValue[0] =  max;
		   returnValue[1] = min ;
		   
		   return returnValue;
	   }
}
