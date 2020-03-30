package net.tc.utils;

import java.util.*;

public class OneToManyHashMap
    extends HashMap
    implements OneToManyMap
{
    public OneToManyHashMap()
    {
    }

    public OneToManyHashMap(int initialCapacity)
    {
        super(initialCapacity);
    }

    public OneToManyHashMap(int initialCapacity, float loadFactor)
    {
        super(initialCapacity, loadFactor);
    }

    public OneToManyHashMap(Map t)
    {
        super(t);
    }

    public boolean contains(Object key, Object value)
    {
        return containsKey(key) && getList(key).contains(value);
    }

    public Iterator iterator(Object key)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).iterator();
    }

    public Object[] toArray(Object key)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).toArray();
    }

    public Object[] toArray(Object key, Object[] a)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).toArray(a);
    }

    public boolean add(Object key, Object value)
    {
        return getOrCreateList(key).add(value);
    }

    public boolean remove(Object key, Object value)
    {
        return containsKey(key) && getList(key).remove(value);
    }

    public boolean containsAll(Object key, Collection c)
    {
        return containsKey(key) && getList(key).containsAll(c);
    }

    public boolean addAll(Object key, Collection c)
    {
        return getOrCreateList(key).addAll(c);
    }

    public boolean addAll(Object key, int index, Collection c)
    {
        return getOrCreateList(key).addAll(index, c);
    }

    public boolean removeAll(Object key, Collection c)
    {
        return containsKey(key) && getList(key).removeAll(c);
    }

    public boolean retainAll(Object key, Collection c)
    {
        return containsKey(key) && getList(key).retainAll(c);
    }

    public boolean equals(Object key, Object value)
    {
        return containsKey(key) && getList(key).equals(value);
    }

    public int size(Object key)
    {
        if(getList(key) != null)
            return getList(key).size();
        return 0;
        //return getOrCreateList(key).size();
    }

    public Object get(Object key)
    {
        if ( size(key) > 0 )
        {
            return get(key, 0);
        }
        else
        {
            return null;
        }
    }

    public Object get(Object key, int index)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).get(index);
    }

    public Object set(Object key, int index, Object element)
    {
        return getOrCreateList(key).set(index, element);
    }

    public Object put(Object key, Object element)
    {
        Object o = get(key);
        remove(key);
        add(key, element);
        return o;
    }

    public List putList(Object key, List list)
    {
        return (List) super.put(key, list);
    }

    public void add(Object key, int index, Object element)
    {
        getOrCreateList(key).add(index, element);
    }

    public Object remove(Object key, int index)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).remove(index);
    }

    public int indexOf(Object key, Object value)
    {
        if ( !containsKey(key) ) return -1;

        return getList(key).indexOf(value);
    }

    public int lastIndexOf(Object key, Object value)
    {
        if ( !containsKey(key) ) return -1;

        return getList(key).lastIndexOf(value);
    }

    public ListIterator listIterator(Object key)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).listIterator();
    }

    public ListIterator listIterator(Object key, int index)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).listIterator(index);
    }

    public List subList(Object key, int fromIndex, int toIndex)
    {
        if ( !containsKey(key) ) return null;

        return getList(key).subList(fromIndex, toIndex);
    }

    private List getOrCreateList(Object key)
    {
        List vals = getList(key);

        if ( vals == null )
        {
            vals = new Vector();
            putList(key, vals);
        }

        return vals;
    }

    public List getList(Object key)
    {
        return (List) super.get(key);
    }
}
