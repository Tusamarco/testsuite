package net.tc.utils;

import java.util.*;

public interface OneToManyMap extends Map
{
    boolean contains(Object key, Object value);
    Iterator iterator(Object key);
    Object[] toArray(Object key);
    Object[] toArray(Object key, Object[] a);
    boolean add(Object key, Object value);
    boolean remove(Object key, Object value);
    boolean containsAll(Object key, Collection c);
    boolean addAll(Object key, Collection c);
    boolean addAll(Object key, int index, Collection c);
    boolean removeAll(Object key, Collection c);
    boolean retainAll(Object key, Collection c);
    boolean equals(Object key, Object value);
    Object get(Object key, int index);
    Object set(Object key, int index, Object element);
    void add(Object key, int index, Object element);
    Object remove(Object key, int index);
    int indexOf(Object key, Object value);
    int lastIndexOf(Object key, Object value);
    ListIterator listIterator(Object key);
    ListIterator listIterator(Object key, int index);
    List subList(Object key, int fromIndex, int toIndex);
    List getList(Object key);
}
