package net.tc.utils ;

/**
 * <p>Title: </p>
 *
 * <p>Description: </p>
 *
 * <p>Copyright: Marco Tusa Copyright (c) 2012 Copyright (c) 2005</p>
 *
 * <p>Company: </p>
 *
 * @author not attributable
 * @version 1.0
 */
public class ObjectHandler
{
    public ObjectHandler()
    {
    }
    public static Object defaultTo(Object o1, Object o2)
    {
        if(o1 != null)
            return o1;

        return o2;

    }
}
