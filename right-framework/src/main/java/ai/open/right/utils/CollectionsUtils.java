package ai.open.right.utils;

import java.util.*;

// Copy From `org.apache.commons.collections4.CollectionsUtils`
public class CollectionsUtils {

    public static <T> List<List<T>> partition(List<T> list, int size) {
        if (list == null) {
            throw new NullPointerException("List can not be empty");
        }
        if (size <= 0) {
            throw new IllegalArgumentException("Size must be greater than 0");
        }
        return new Partition<>(list, size);
    }

    public static <T> Map<String, T> merge(Map<String, T> t, Map<String, T> s) {
        if (t == null) {
            return s;
        }
        Map<String, T> r = new LinkedHashMap<String, T>(t);
        if (s != null) {
            for (String k : s.keySet()) {
                r.putIfAbsent(k, s.get(k));
            }
        }
        return r;
    }

    public static <T> List<T> merge(List<T> t, List<T> s) {
        if (t == null) {
            return s;
        }
        List<T> r = new ArrayList<T>(t);
        if (s != null) {
            r.addAll(s);
        }
        return r;
    }

    public static class Partition<T> extends AbstractList<List<T>> {

        protected final List<T> list;
        protected final int size;

        public Partition(final List<T> list, final int size) {
            this.list = list;
            this.size = size;
        }

        @Override
        public List<T> get(final int index) {
            final int listSize = size();
            if (index >= listSize) {
                throw new IndexOutOfBoundsException("Index " + index + " must be less than size " + listSize);
            }
            final int start = index * size;
            final int end = Math.min(start + size, list.size());
            return list.subList(start, end);
        }

        @Override
        public int size() {
            return (int) Math.ceil((double) list.size() / (double) size);
        }

        @Override
        public boolean isEmpty() {
            return list.isEmpty();
        }
    }
}
