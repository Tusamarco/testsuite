package net.tc.utils;
import java.lang.ref.SoftReference;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.text.ParseException;
import java.util.Date;
import java.util.Calendar;
import java.util.GregorianCalendar;

/**
 * <p>Title: </p>
 *
 * <p>Description: </p>
 *
 * <p>Copyright: Marco Tusa Copyright (c) 2012 Copyright (c) 2007</p>
 *
 * <p>Company: </p>
 *
 * @author not attributable
 * @version 1.0
 */
public class TimeTools {
    private static String dayFormat = "yyyy-MM-dd";
    private static String timeFormat = "HH:mm:ss";

    public static String getDayFormat()
    {
        return dayFormat;
    }
    public static String getTimeFormat()
    {
        return timeFormat;
    }

    public TimeTools() {
    }
    public static String GetFullDate(long systemTime)
    {
        java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat(
                getDayFormat() + "_" + getTimeFormat() );

        try {
            return sdf.format(systemTime);
        } catch (java.lang.NullPointerException npEx) {
            System.out.println(npEx);
            return "";
        }

    }

    public static String GetCurrentDay() {
        java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat(
                getDayFormat());

        try {
            return sdf.format(java.util.Calendar.getInstance().getTime());
        } catch (java.lang.NullPointerException npEx) {
            System.out.println(npEx);
            return "";
        }
    }
    public static String GetCurrentTime() {
     java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat( getTimeFormat() );

     try
     {
         return sdf.format( java.util.Calendar.getInstance().getTime() );
     }
     catch( java.lang.NullPointerException npEx )
     {
         System.out.println(npEx);
         return "";
     }
    }

	public static String GetCurrentTime(Calendar cal) {
	    java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat( getTimeFormat() );
	
	    try
	    {
	        return sdf.format( cal.getTime() );
	    }
	    catch( java.lang.NullPointerException npEx )
	    {
	        System.out.println(npEx);
	        return "";
	    }
	   }

 public static String GetCurrent()
 {
     java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat( dayFormat + " " + timeFormat );

     try
     {
         return sdf.format( java.util.Calendar.getInstance().getTime() );
     }
     catch( java.lang.NullPointerException npEx )
     {
         System.out.println(npEx);
         return "";
     }


 }

 public static String GetCurrent(Calendar calendar)
 {
     java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat( dayFormat + " " + timeFormat );

     try
     {
         return sdf.format( calendar.getTime() );
     }
     catch( java.lang.NullPointerException npEx )
     {
         System.out.println(npEx);
         return "";
     }


 }

 public static Date getDate(String date, String format)
 {

     DateFormat df = new SimpleDateFormat(format);

       try
       {
           Date dateR = df.parse(date);
           return dateR;
       } catch (ParseException e)
       {
           e.printStackTrace();
       }


     return null;
 }

 
 public static String getYear()
 {
     GregorianCalendar calendar = new GregorianCalendar();
     String year = new Integer(calendar.get(GregorianCalendar.YEAR)).toString();
     return year;
 }

 public static String getYear(Calendar calendar)
 {
     String year = new Integer(calendar.get(GregorianCalendar.YEAR)).toString();
     return year;
 }

 
 public static String getMonth()
 {
     Calendar calendar = new GregorianCalendar();
     int m = calendar.get(GregorianCalendar.MONTH);
     switch (m)
     {
         case 0:return "Jenuary";
         case 1:return "February";
         case 2:return "March";
         case 3:return "April";
         case 4:return "May";
         case 5:return "June";
         case 6:return "July";
         case 7:return "August";
         case 8:return "September";
         case 9:return "October";
         case 10:return "November";
         case 11:return "December";
     }
     return null;
    }
 public static String getMonth(Calendar calendar )
 {
     int m = calendar.get(GregorianCalendar.MONTH);
     switch (m)
     {
         case 0:return "Jenuary";
         case 1:return "February";
         case 2:return "March";
         case 3:return "April";
         case 4:return "May";
         case 5:return "June";
         case 6:return "July";
         case 7:return "August";
         case 8:return "September";
         case 9:return "October";
         case 10:return "November";
         case 11:return "December";
     }
     return null;
    }

 
 public static String getDayNumber()
    {
      Calendar calendar = new GregorianCalendar();
      int d = calendar.get(GregorianCalendar.DAY_OF_MONTH);
      String day = new Integer(d + 1).toString();
      if(day.length()<2)
          day = "0"+day;
      return day;
  }

 public static String getDayNumber(Calendar calendar)
 {
   int d = calendar.get(GregorianCalendar.DAY_OF_MONTH);
   String day = new Integer(d + 1).toString();
   if(day.length()<2)
       day = "0"+day;
   return day;
}

 public static Calendar getCalendar(String dateTime){
	if(dateTime == null || dateTime.equals(""))
	  return null;
	
	Date byThisDate = TimeTools.getDate(dateTime, dayFormat + " " + timeFormat );
	Calendar cal = Calendar.getInstance();
	cal.setTime(byThisDate);
	return cal;
	
  }
  
  
  public static Calendar getCalendarFromCalendarDateAddDays(Calendar cal, int  numberOfdaysFromCalendarDate){
	if(cal != null ){
	  
	  Calendar newCal = (Calendar)cal.clone();
	  SoftReference sf = new SoftReference((Calendar)cal.clone());
	  ((Calendar) sf.get()).add(Calendar.DATE, numberOfdaysFromCalendarDate);
	  
	  return ((Calendar) sf.get());
	}
	
	return null;
	
	
  }
  public static String getTimeStamp(long systemTime){
      return getTimeStamp(systemTime, "yyyy-MM-dd hh:mm:ss");
 }

  public static String getTimeStamp(long systemTime, String format){
	 if(systemTime == 0)
	     return  null;
	 
	    Date dNow = new Date(systemTime);
	      SimpleDateFormat ft = 
	      new SimpleDateFormat (format);
	
	     return ft.format(dNow);
	
	
	}
  public static String getTimeStamp(Calendar cal, String format){
	 if(cal == null)
	     return  null;
	 
	    Date dNow = cal.getTime();
	      SimpleDateFormat ft = 
	      new SimpleDateFormat (format);
	
	     return ft.format(dNow);
	
	
	}

  public static String getTimeStampFromDate(Date date, String format){
	 if(date == null)
	     return  null;
	 
	 if(format == null)
		 format =  getDayFormat() + " " + getTimeFormat();
	 
	    Date dNow = date;
	      SimpleDateFormat ft = 
	      new SimpleDateFormat (format);
	
	     return ft.format(dNow);
	
	
	}

}
