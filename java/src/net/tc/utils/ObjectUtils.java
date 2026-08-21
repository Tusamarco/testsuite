package net.tc.utils;

/**
 * <p>Title: </p>
 *
 * <p>Description: </p>
 *
 * <p>Copyright: Marco Tusa Copyright (c) 2012 Copyright (c) 2004</p>
 *
 * <p>Company: </p>
 *
 * @author not attributable
 * @version 1.0
 */
public class ObjectUtils
{
	public ObjectUtils()
	{
	}
	/**
 * Checks for null and return def if found.
 * @param o
 * @param def
 * @return String If null, default.  Otherwise String value of object
 */
public static Object defaultTo(Object o, Object def)
{
	if(o == null)
		return def;
	else
		return o;
}

}
