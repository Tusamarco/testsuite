package net.tc.utils;

import java.util.*;
import java.io.ObjectOutputStream;
import java.io.ObjectInputStream;
import java.io.IOException;


/**
 * This class exist because the "normal" implementation of the Map interface has a serious BUG
 * It return as Hash code the sum of the hash code that the map itself contains that means that
 * it is possible to have more then one hashcode with the same value. The effect is that if you
 * load a map inside another map the new Map could  replace an existing object also if the keys
 * are different.
 *
 * More: this map implementation is using vectors internally that are automatically synchronized
 */
public class SynchronizedMap extends TreeMap<Object, Object>
{
  /**
	 * 
	 */
  private static final long serialVersionUID = 1L;

  private Vector<Object> keys = null;
  private Vector<Object> values = null;

  
  public SynchronizedMap()
  {
	super();
	keys = new Vector<Object>(0,1);
	values = new Vector<Object>(0,1);

  }
  
  public SynchronizedMap(int initialCapacity)
  {
	 super();
	keys = new Vector<Object>(initialCapacity,1);
	values = new Vector<Object>(initialCapacity,1);
  }
  public SynchronizedMap(int initialCapacity, int loadFactor)
  {
	  super();
	keys = new Vector<Object>(initialCapacity,loadFactor);
	values = new Vector<Object>(initialCapacity,loadFactor);
  }
  public SynchronizedMap(Map<?, ?> m)
  {
	  super();
	keys = new Vector<Object>(0,1);
	values = new Vector<Object>(0,1);
	putAll(m);
  }


  //private Map internalValues = new HashMap();
  public static Boolean isMap = null;
  
  @Override
  public Collection values() {
	return values;
	  
  }

  //MAP methods
//@Override
//  public int hashCode()
//  {
//	if( keys == null && values == null )
//	  return 0;
//
//	return 0;
//	//keys.hashCode();
//  }
@Override
  public int size()
  {
	if( keys == null && values == null  )
	  return 0;

	return keys.size();
  }

@Override
  public void clear()
  {
	if( keys == null && values == null)
	  return;

	keys.clear();
	values.clear();
  }

@Override
  public boolean isEmpty()
  {
	if( keys == null && values == null  )
	  return true;
	return keys.isEmpty();
  }

@Override
  public boolean containsKey( Object key )
  {
	if( keys == null )
	  return false;

	return keys.contains( key );
  }


  public boolean containsSubKey( Object value, int pos )
  {
	if( keys == null || keys.size() <= 0)
	  return false;
	if(keys.get(0) instanceof SynchronizedMap)
	{
	  if(((SynchronizedMap)keys.get(0)).keys.size() >= pos)
	  {
		for( int i = 0; i < keys.size(); i++ )
		{
			SynchronizedMap k = ( SynchronizedMap ) keys.get( i );
		  for(int ik = 0 ; ik < k.size() ; ik++)
		  {
			if( k.values.get( ik ) != null && k.values.get( ik ).equals( value ) )
			  return true;
		  }

		}
	  }
	}

	return false;
  }

  @Override
  public boolean containsValue( Object value )
  {
	if( values == null )
	  return false;

	return values.contains( value );
  }
 
@Override
public void putAll( Map t )
  {
	if( t == null )
	  return;
	try{
	  Iterator<?> it = t.keySet().iterator();
	  while( it.hasNext() )
	  {
		Object tk = it.next();
		Object tv = t.get( tk );
		put( tk, tv );
	  }
	}catch(Exception ex)
	{ex.printStackTrace();}

  }
 
@Override
  public Set keySet()
  {
	if(keys != null)
	  return new HashSet<Object>((Collection<Object>)keys.subList(0,keys.size()));
	return null;
  }

@Override
  public Set entrySet()
  {
	if(keys != null)
		return new HashSet<Object>((Collection<Object>)values.subList(0,values.size()));
	return null;
  }

@Override
  public Object get( Object key )
  {
	if( key == null || keys == null || values == null)
	  return null;

	if(keys.contains(key))
	{
	  int index = keys.indexOf(key);
	  return values.get(index);
	}
	return null;
  }

 
@Override
  public synchronized Object remove( Object key )
  {
	if(  key == null || keys == null || values == null )
	  return null;
	try
	{
	  if(keys.contains(key))
	  {
		int index = keys.indexOf( key );
		keys.remove(index);
		values.remove(index);
		return null;
	  }

	  return null;
	}
	catch( Exception ex )
	{
	  ex.printStackTrace();
	}
	return null;
  }
  /**
   * Insert an object into the collection
   */

@Override
   public synchronized Object  put( Object tk, Object tv )
  {
	if(  tk == null || keys == null || values == null )
	  return null;

	if(keys.contains(tk))
	  remove(tk);
	keys.add( tk );
	int index = keys.indexOf( tk );
	values.add( index, tv );


	return new Integer(index);
  }



   //*************************** NOT MAP Methods
   
	public Object getKeyByPosition(int pos){
		 return keys.get(pos);
	   }


	public synchronized Object getValueByPosition(int pos){
		 return values.get(pos);
	   }
   
   
   public List<Object> get(Object inkey, int pos)
   {
 	List<Object> l = null;
 	if( keys == null || keys.size() <= 0)
 	  return null;
 	if(keys.get(0) instanceof SynchronizedMap)
 	{
 	  if(((SynchronizedMap)keys.get(0)).keys.size() >= pos)
 	  {
 		l = new ArrayList<Object>(0);
 		for( int i = 0; i < keys.size(); i++ )
 		{
 		  SynchronizedMap k = ( SynchronizedMap ) keys.get( i );
 		  if(k.size() >= pos)
 		  {
 			if(k.values.get(pos) != null && k.values.get(pos).equals(inkey))
 			{
 			  Object o = values.get(i);
 			  l.add(o);
 			}

 		  }
 		}
 	  }
 	}
 	return l;
   }

   
   public int getAsInt( Object key )
   {
 	if( key == null || keys == null || values == null)
 	  return 0;

 	try
 	{
 	  if( get( key ) instanceof Integer )
 		return( ( Integer ) get( key ) ).intValue();

 	  int i = Integer.parseInt( get( key ).toString() );
 	  return i;
 	}
 	catch( Exception ex )
 	{
 	  ex.printStackTrace();
 	}
 	return 0;

   }


   public String getAsString( Object key )
   {
 	if(  key == null || keys == null || values == null)
 	  return null;

 	try
 	{
 	  return get( key ).toString();
 	}
 	catch( Exception ex )
 	{
 	  ex.printStackTrace();
 	}
 	return null;
   }
   
   
   private void writeObject( ObjectOutputStream oos ) 
   {
	 try {
		oos.defaultWriteObject();
	} catch (IOException e) {
		// TODO Auto-generated catch block
		e.printStackTrace();
	}
   }


   private void readObject( ObjectInputStream ois )
   {
	 try {
		ois.defaultReadObject();
	} catch (ClassNotFoundException | IOException e) {
		// TODO Auto-generated catch block
		e.printStackTrace();
	}
   }


   public synchronized Object[] getValuesAsArrayOrderByKey(){
	 if(keys !=null && keys.size()>0){
	   Object[] keys = getKeyasOrderedArray();
	   Object[] values = new Object[keys.length];
	   int i =0;
	   for(Object o : keys){
		 values[i++]= this.get(o);
	   }
	   return values;
	 }
	 return null;
   }

   
   public Object[] getValuesAsArrayOrderByKey(Object[] keys){
	 if(keys !=null && keys.length>0){
	   Object[] values = new Object[keys.length];
	   int i =0;
	   for(Object o : keys){
		 values[i++]= this.get(o);
	   }
	   return values;
	 }
	 return null;
   }


   public Object[] getKeyasOrderedArray()
   {
	 if(keys == null)
	   return null;

	 Object[] oA = new Object[keys.size()];
	 for(int i = 0 ; i < keys.size() ; i++)
	 {
	   if(i < keys.size())
		   oA[i] = keys.get(i);
	 }
	 return oA;
   }

   public String[] getKeyasOrderedStringArray()
   {
	 if(keys == null)
	   return null;

	 String[] oA = new String[keys.size()];
	 for(int i = 0 ; i < keys.size() ; i++)
	 {
	   oA[i] = (String)keys.get(i).toString();
	 }
	 return oA;
   }

   public String getKeyasUnorderdString()
   {
	 if(keys == null)
	   return null;

	 StringBuffer sb = new StringBuffer();
	 for(int i = 0 ; i < keys.size() ; i++)
	 {
	   sb.append(i>0?",":"");
	   sb.append((String)keys.get(i).toString());
	 }
	 return sb.toString();
   }

   
   public static <K, V extends Comparable<? super V>> SortedSet<Map.Entry<K, V>> entriesSortedByValues(Map<K, V> map) {
	 SortedSet<Map.Entry<K, V>> sortedEntries = new TreeSet<Map.Entry<K, V>>(
		 new Comparator<Map.Entry<K, V>>() {
		   @Override
		   public int compare(Map.Entry<K, V> e1, Map.Entry<K, V> e2) {
			 return e1.getValue().compareTo(e2.getValue());
		   }
		 });
	 sortedEntries.addAll(map.entrySet());
	 return sortedEntries;
   }

   
   public Iterator<Object> iterator()
   {
 	if(keys == null)
 	  return null;

 	return keys.iterator();
   }

   public void putAll(Object key, Map<?, ?> t )
   {
 	if( t == null )
 	  return;
 	try{
 	  Map<Object, Object> tMap = new SynchronizedMap(t.size());
 	  Iterator<?> it = t.keySet().iterator();
 	  while( it.hasNext() )
 	  {
 		Object tk = it.next();
 		Object tv = t.get( tk );
 		tMap.put(tk, tv);
 	  }
 	  put( key, tMap );

 	}catch(Exception ex)
 	{ex.printStackTrace();}

   }
   
   public boolean containsInternalValue( Object value )
   {
 	if( values == null )
 	  return false;

 	for(int i =0 ; i < values.size(); i++)
 	{
 	  if(value instanceof SynchronizedMap)
 	  {
 		if(((SynchronizedMap)values.get(i)).values.equals((SynchronizedMap)value))
 		  return true;
 	  }
 	  else
 	  {
 		if(values.get(i).equals(value))
 		  return true;
 	  }
 	}
 	return false;
   }

   
   public boolean containsInternalKey( Object key )
   {
 	if( keys == null )
 	  return false;

 	for(int i =0 ; i < values.size(); i++)
 	{
 	  if(key instanceof SynchronizedMap)
 	  {
 		if(((SynchronizedMap)keys.get(i)).values.equals(((SynchronizedMap)key).values))
 		  return true;
 	  }
 	  else
 	  {
 		if(keys.get(i).equals(key))
 		  return true;
 	  }
 	}
 	return false;
   }
   
   
   public Object getInternalKey( Object key )
   {
 	if( keys == null )
 	  return null;

 	for(int i =0 ; i < values.size(); i++)
 	{
 	  if(key instanceof SynchronizedMap)
 	  {
 		if(((SynchronizedMap)keys.get(i)).values.equals(((SynchronizedMap)key).values))
 		  return values.get(i);
 	  }
 	  else
 	  {
 		if(keys.get(i).equals(key))
 		  return values.get(i);
 	  }
 	}
 	return null;
   }
   
   @Override
   public String toString() {
	  return  this.getClass().getName();	   
	   
//	   StringBuffer sb = new StringBuffer();
//	   
//	   Iterator it = this.keySet().iterator();
//	   while(it.hasNext()) {
//		   sb.append(it.toString() + ":");
//		   
//		   
//	   }
//	   return sb.toString();
	   
	   
   }
   
}
