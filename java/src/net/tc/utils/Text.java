package net.tc.utils;

import java.util.StringTokenizer;
import java.util.List;
import java.util.Vector;
import java.util.ResourceBundle;
import java.io.UnsupportedEncodingException;
import java.nio.charset.CharacterCodingException;
import java.io.ByteArrayInputStream;
import java.io.InputStreamReader;
import java.nio.charset.Charset;
import java.nio.charset.CharsetDecoder;
import java.nio.charset.CharsetEncoder;
import java.nio.ByteBuffer;
import java.nio.CharBuffer;
import java.io.InputStream;
//import java.io.StringBufferInputStream;
import java.io.ObjectInputStream;
import java.io.StringWriter;
import java.io.PrintWriter;
import java.io.ByteArrayOutputStream;
import java.io.BufferedWriter;
import java.io.OutputStreamWriter;
import java.io.Writer;

/**
 * Utility class for common String operations.  Methods that take Object as a parameter
 * will extract the String representation of the object by either casting to
 * String or by calling the Object's toString() method.
 *
 * <p>Title: ISMA</p>
 * <p>Description: </p>
 * <p>Copyright: Marco Tusa Copyright (c) 2012 Copyright (c) 2003</p>
 * <p>Company: TusaCentral</p>
 * @author  Tusa
 * @version 1.0
 */
public class Text
{
//	private final static Charset[] source = { Charset.forName("ISO-8859-1"),
//											  Charset.forName("EUC-CN"),
//											  Charset.forName("windows-1256"),
//											  Charset.forName("Shift-JIS") };

    /**
     * Checks if String is null or 0 length
     */
    public static boolean isEmpty(Object o)
    {
        return o==null || toString(o).length()==0;
    }

    /**
     * Checks that entire array has "empty" Strings (null or 0 length)
     * @param s
     * @return
     */
    public static boolean isEmpty(String[] s)
    {
        if ( s == null )
        {
            return true;
        }

        for (int i = 0; i < s.length; i++)
        {
            if (!isEmpty(s[i]))
            {
                return false;
            }
        }

        return true;
    }

    public static boolean isBlank(String[] s)
    {
//        Object[] o = new Object[s.length];
//        System.arraycopy(s, 0, o, 0, s.length);
        return isBlank((Object[]) s);
    }

    /**
     * Checks that entire array has "blank" Strings (null or whitespace)
     * @param s
     * @return
     */
    public static boolean isBlank(Object[] s)
    {
        if ( s == null )
        {
            return true;
        }

        for (int i = 0; i < s.length; i++)
        {
            if (!isBlank(s[i]) && !s[i].equals("") && !s[i].equals(" "))
            {
                return false;
            }
        }

        return true;
    }

    /**
     * Checks if String is null or contains only whitespace
     */
    public static boolean isBlank(Object o)
    {
        return o==null || toString(o).trim().length()==0;
    }

    /**
     * Checks for null and return def if found.
     * @param o
     * @param def
     * @return String If null, default.  Otherwise String value of object
     */
    public static String defaultTo(Object o, Object def)
    {
        if(o == null)
            return toString(def);
        else if (o instanceof String && ((String)o).equals(""))
            return toString(def);
        else
            return toString(o);
//        return o==null ? toString(def) : toString(o);
    }

    /**
     * Convert a String to an int.  If null or invalid, default value is returned.
     * @param val
     * @param deflt
     * @return
     */
    public static int toInt(Object val, int deflt)
    {
        if ( val == null )
        {
            return deflt;
        }

        try
        {
            return Integer.parseInt(toString(val));
        }
        catch ( NumberFormatException e )
        {
            return deflt;
        }
    }

    /**
     * Convert a String to an Integer.  If invalid, default value is returned.
     * @param val
     * @param deflt
     * @return
     */
    public static Integer toInteger(Object val, Integer deflt)
    {
        if ( val == null )
        {
            return null;
        }

        try
        {
            return new Integer(toString(val));
        }
        catch ( NumberFormatException e )
        {
            return deflt;
        }
    }

    /**
     * Convert a String to a long.  If null or invalid, default value is returned.
     * @param val
     * @param deflt
     * @return
     */
    public static long toLng(Object val, long deflt)
    {
        if ( val == null )
        {
            return deflt;
        }

        try
        {
            return Long.parseLong(toString(val));
        }
        catch ( NumberFormatException e )
        {
            return deflt;
        }
    }

    /**
     * Convert a String to a Long.  If invalid, default value is returned.
     * @param val
     * @param deflt
     * @return
     */
    public static Long toLong(Object val, Long deflt)
    {
        if ( val == null )
        {
            return null;
        }

        try
        {
            return new Long(toString(val));
        }
        catch ( NumberFormatException e )
        {
            return deflt;
        }
    }

    /**
     * Defaults to "" if String is null.
     * @param o
     * @return
     */
    public static String defaultToEmpty(Object o)
    {
        return defaultTo(o, "");
    }

    /**
     * @param arr array of values
     * @param delimiter delimiter to use to separate values
     * @return concatented list of values using specified delimiter
     */
    public static String delimit(Object[] arr, String delimiter)
    {
        return delimit(arr, delimiter, "", "");
    }

    public static String delimit(Object[] arr, String delimiter, String prefix, String suffix)
    {
        StringBuffer sb = new StringBuffer();

        for (int i = 0; i < arr.length; i++)
        {
            if ( i > 0 )
            {
                sb.append(delimiter);
            }

            sb.append(prefix);
            sb.append(Text.toString(arr[i]));
            sb.append(suffix);
        }

        return sb.toString();
    }

    /**
     * String replace.  For use with older versions of JDK that do not support
     * String.replace().  Can be later replaced with API method.
     * @param s
     * @param s1
     * @param s2
     * @return
     */
    public static String replace(String s, String s1, String s2)
    {
        int i = 0;

        if( s1.length() == 0 )
        {
            return s;
        }

        StringBuffer sb = new StringBuffer();
        do
        {
            int j = s.indexOf(s1, i);

            if (j == -1)
            {
                sb.append(s.substring(i));
                return sb.toString();
            }
            sb.append(s.substring(i, j));
            sb.append(s2);
            i = j + s1.length();
        }
        while( i != s.length() );

        return sb.toString();
    }

    /**
     * Removes accented characters and replace with unaccented equivalent.
     * @param s
     * @return
     */
	public static String removeAccents(Object o)
	{
		if ( o==null ) return null;

		String s = toString(o).toUpperCase();

//		char[] in =
//			{'Ä', 'À', '�\uFFFD', 'Â', 'Ã', 'Ë', 'Ê', 'É', 'È', '�\uFFFD', 'Î', '�\uFFFD', 'Ì', 'Ö', 'Ô', 'Ó', 'Ò', 'Õ', 'Ü', 'Û', 'Ú', 'Ù', 'Ñ'};
//		char[] outc =
//			{
//			'A', 'A', 'A', 'A', 'A', 'E', 'E', 'E', 'E', 'I', 'I', 'I', 'I', 'O', 'O', 'O', 'O', 'O', 'U', 'U', 'U', 'U', 'N'};
//
//		for (int i = 0; i < in.length; i++)
//		{
//			s = s.replace(in[i], outc[i]);
//		}

		return s;
	}

    /**
     * Extracts String representation of object by either casting or calling
     * its toString() method.  If null, default is returned.
     * @param o
     * @param def
     * @return
     */
    public static String toString(Object o, String def)
    {
        if ( o == null )
        {
            return def;
        }
        else if (o instanceof String)
        {
            return (String) o;
        }
        else
        {
            return o.toString();
        }

    }

    /**
     * Extracts String representation of object by either casting or calling
     * its toString() method.  Returns null if object is null.
     * @param o
     * @return
     */
    public static String toString(Object o)
    {
        return toString(o, null);
    }

    public static String toProperCase(Object o)
    {
        if ( o == null ) return null;
        String s = toString(o).trim();
        StringBuffer sb = new StringBuffer();
        if ( s.length() > 0 )
        {
            sb.append(s.substring(0, 1).toUpperCase());
        }
        if ( s.length() > 1 )
        {
            sb.append(s.substring(1).toLowerCase());
        }

        return sb.toString();
    }

    public static String[] castToStringArray(Object[] arr)
    {
        String[] arr2 = new String[arr.length];
        System.arraycopy(arr, 0, arr2, 0, arr.length);
        return arr2;
    }

    public static String[] getMultipleResources(ResourceBundle rsc, String[] keys)
    {
        List l = new Vector();
        for (int i = 0; i < keys.length; i++)
        {
            l.add(rsc.getString(keys[i]));
        }
        return (String[]) l.toArray(new String[0]);
    }

    public static String trimTo(String s, int maxChars)
    {
        if ( s == null || s.length() <= maxChars )
            return s;
        else
            return s.substring(0, maxChars);
    }

    public static boolean isNumeric(Object o)
    {
        if ( o == null )
            return false;

        try
        {
            Long.valueOf(toString(o));
            return true;
        }
        catch (NumberFormatException ex)
        {
            return false;
        }
    }

	public static String getEncodedString( Object obj, String encoding )
	{

		if( obj instanceof String )
		{
			try
			{

				ByteArrayOutputStream bos = new ByteArrayOutputStream();
				Writer outs = new OutputStreamWriter( bos, encoding );
				outs.write( ( String ) obj );
				outs.flush();
				return bos.toString();
			}
			catch( Exception ex )
			{
				return null;
			}
		}
		else
		{
			try
			{

				ByteArrayOutputStream bos = new ByteArrayOutputStream();
				Writer outs = new OutputStreamWriter( bos, encoding ) ;
				outs.write( obj.toString() );
				outs.flush();
				return bos.toString();
			}
			catch( Exception ex )
			{
				return null;
			}
		}

	}
//	public static String encodeToUtf (String str, int index) {
//
//   CharBuffer isoChars = null;
//
//	try {
//		 Charset utf8_charset = Charset.forName("UTF-8");
//		 CharsetEncoder encoder = utf8_charset.newEncoder();
//		 byte[] bytes = str.getBytes();
//		 ByteBuffer isoBytes = ByteBuffer.wrap(bytes);
//
//		 switch(index) {
//				case 1: // For English, French and Spanish use ISO-8859-1
//						isoChars = source[0].decode(isoBytes);
//						break;
//				case 2:
//						// For Chinese use EUC-CN encoding
//						isoChars = source[1].decode(isoBytes);
//						break;
//				case 3:
//						// For Arabic use windows-1256 encoding
//						isoChars = source[2].decode(isoBytes);
//						break;
//				case 4:
//						// For Japanese use EUC-JP encoding
//						isoChars = source[3].decode(isoBytes);
//						break;
//		 }
//		 str = isoChars.toString();
//	}
//	catch (Exception ex) {
//	}
//
//  return str;
//}

	public static String castToUTFNio(String str, String sourceEncoding) throws UnsupportedEncodingException, CharacterCodingException
	  {
		  String utf = null;
		  if(sourceEncoding.equals("UTF-8"))
			  return str;

		  byte[] br = str.getBytes(sourceEncoding);
		  ByteArrayInputStream bai = new ByteArrayInputStream(br);
		  InputStreamReader impR = new InputStreamReader(bai, sourceEncoding);

//		  System.out.println("***** " + impR.getEncoding());

		  Charset csFrom = Charset.forName(impR.getEncoding());
		  Charset csTo = Charset.forName("UTF-8");
		  CharsetDecoder decoder = csFrom.newDecoder();
		  CharsetEncoder encoder = csTo.newEncoder();

		  ByteBuffer bb = ByteBuffer.wrap(br);
		  CharBuffer cbFrom = decoder.decode(bb);
		  ByteBuffer bbTo = encoder.encode(cbFrom);

		  utf = new String(bbTo.array());
		return utf;
	}
	public static String castToISONio(String str, String sourceEncoding) throws UnsupportedEncodingException, CharacterCodingException
	  {
		  String utf = null;
		  if(sourceEncoding.equals("ISO-8859-1"))
			  return str;
		  byte[] br = str.getBytes(sourceEncoding);
		  ByteArrayInputStream bai = new ByteArrayInputStream(br);
		  InputStreamReader impR = new InputStreamReader(bai, sourceEncoding);

//		  System.out.println("***** " + impR.getEncoding());

		  Charset csFrom = Charset.forName(impR.getEncoding());
		  Charset csTo = Charset.forName("ISO-8859-1");
		  CharsetDecoder decoder = csFrom.newDecoder();
		  CharsetEncoder encoder = csTo.newEncoder();

		  ByteBuffer bb = ByteBuffer.wrap(br);
		  CharBuffer cbFrom = decoder.decode(bb);
		  ByteBuffer bbTo = encoder.encode(cbFrom);

		  utf = new String(bbTo.array());
		return utf;
	}
	public static String castToUTF(String iso_str)
	{
		try
		{
			String cstr = null;
			if( iso_str != null )
			{
				byte[] b = iso_str.getBytes( "ISO-8859-1" );
				cstr = new String( b, "UTF-8" );
			}
			return cstr;
		}
		catch(Exception ex)
		{}
		return iso_str;
	}

	public static String getUTFStringFromDb(String str, String lang)
	{
		if(str == null || lang == null || str.equals("") || lang.equals(""))
			return "";
			CharBuffer isoChars = null;
//			ByteBuffer utfBytes = null;

			try
			{
//				Charset utf8_charset = Charset.forName( "UTF-8" );
//				CharsetEncoder encoder = utf8_charset.newEncoder();
				byte[] bytes = str.getBytes();
				ByteBuffer isoBytes = ByteBuffer.wrap( bytes );
				if(lang.toLowerCase().equals("ar"))
				{
					isoChars = Charset.forName( "windows-1256" ).decode( isoBytes );
				}
				else if(lang.toLowerCase().equals("zh"))
				{
					isoChars = Charset.forName( "EUC-CN" ).decode( isoBytes );
				}
				else if(lang.toLowerCase().equals("fr") || lang.toLowerCase().equals("en") || lang.toLowerCase().equals("es"))
				{
					isoChars = Charset.forName( "ISO-8859-1" ).decode( isoBytes );
				}
				else
					return str;

							}
			catch( Exception ex )
			{
				ex.printStackTrace();
			}

			return isoChars.toString();

	}
    public static String capitalize(String s)
    {
        String[] sa = null;
        StringBuffer sb = new StringBuffer();
        if(s.indexOf(" ") > 0)
            sa = s.split(" ");
        else
            sa = new String[]{s};

        for(int i = 0 ; i < sa.length ; i++)
            sb.append(sa[i].substring(0,1).toUpperCase()+sa[i].substring(1));
        return sb.toString();
    }

    public static String getAsStringSeparated(List l , String s)
    {
        if(l == null || s == null)
            return "";

        String s2Return ="";

        for(int il = 0 ;il < l.size() ; il++)
        {
            if(l.get(il) instanceof String )
                s2Return = s2Return + (String) l.get(il) + s;
            else if(l.get(il) instanceof Long || l.get(il) instanceof Integer)
                s2Return = s2Return + String.valueOf(l.get(il)) + s;

        }


        return s2Return;
    }
}
