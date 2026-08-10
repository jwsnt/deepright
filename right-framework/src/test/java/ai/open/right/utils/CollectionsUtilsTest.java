
package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CollectionsUtilsTest {

    @Test(expected = IllegalArgumentException.class)
    public void testEmpty() {
        Assert.assertTrue(CollectionsUtils.partition(new ArrayList<>(), 0).isEmpty());
    }

    @Test(expected = NullPointerException.class)
    public void testNullPoint() {
        CollectionsUtils.partition(null, 10);
    }

    @Test(expected = IndexOutOfBoundsException.class)
    public void testOverSize() {
        CollectionsUtils.partition(Arrays.asList("A", "B", "C"), 2).get(10);
    }

    @Test
    public void testPartition() {
        CollectionsUtils.Partition partition = new CollectionsUtils.Partition(Arrays.asList("A", "B", "C"), 2);
        Assert.assertFalse(partition.isEmpty());
    }

    @Test
    public void testPartitionEmptyList() {
        List<List<Object>> list = CollectionsUtils.partition(new ArrayList<>(), 1);
        Assert.assertTrue(list.isEmpty());
        Assert.assertEquals(0, list.size());
    }

    @Test
    public void testPartitionExactSize() {
        List<List<String>> list = CollectionsUtils.partition(Arrays.asList("A", "B"), 2);
        Assert.assertEquals(1, list.size());
        Assert.assertEquals(Arrays.asList("A", "B"), list.get(0));
    }

    @Test
    public void testPartitionSmallList() {
        List<List<String>> list = CollectionsUtils.partition(Arrays.asList("A"), 2);
        Assert.assertEquals(1, list.size());
        Assert.assertEquals(Arrays.asList("A"), list.get(0));
    }

    @Test
    public void testMergeMapNull() {
        Assert.assertNull(CollectionsUtils.merge((Map<String, Object>) null, null));
    }

    @Test
    public void testMergeMapEmpty() {
        Map<String, Object> result = CollectionsUtils.merge(new HashMap<>(), new HashMap<>());
        Assert.assertTrue(result.isEmpty());
    }

    @Test
    public void testMergeListNull() {
        Assert.assertNull(CollectionsUtils.merge((List<Object>) null, null));
    }

    @Test
    public void testMergeListEmpty() {
        List<Object> result = CollectionsUtils.merge(new ArrayList<>(), new ArrayList<>());
        Assert.assertTrue(result.isEmpty());
    }

    @org.junit.jupiter.api.Test
    public void testSplitBoundary() {
        java.util.List<Integer> list = java.util.Arrays.asList(1, 2, 3);
        java.util.List<java.util.List<Integer>> split = CollectionsUtils.partition(list, 5);
        org.junit.jupiter.api.Assertions.assertEquals(1, split.size());
        org.junit.jupiter.api.Assertions.assertEquals(3, split.get(0).size());
    }

    @org.junit.jupiter.api.Test
    public void testSplitSingle() {
        java.util.List<Integer> list = java.util.Arrays.asList(1, 2, 3);
        java.util.List<java.util.List<Integer>> split = CollectionsUtils.partition(list, 1);
        org.junit.jupiter.api.Assertions.assertEquals(3, split.size());
    }

    @org.junit.jupiter.api.Test
    public void testPartitionNullListJunit5() {
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            CollectionsUtils.partition(null, 10);
        });
    }

    @org.junit.jupiter.api.Test
    public void testPartitionInvalidSizeJunit5() {
        org.junit.jupiter.api.Assertions.assertThrows(IllegalArgumentException.class, () -> {
            CollectionsUtils.partition(new ArrayList<>(), -1);
        });
    }

    @org.junit.jupiter.api.Test
    public void testMergeMapFirstNullJunit5() {
        Map<String, String> s = new HashMap<>();
        s.put("key", "value");
        Map<String, String> result = CollectionsUtils.merge(null, s);
        org.junit.jupiter.api.Assertions.assertEquals(s, result);
    }

    @org.junit.jupiter.api.Test
    public void testMergeMapSecondNullJunit5() {
        Map<String, String> t = new HashMap<>();
        t.put("key", "value");
        Map<String, String> result = CollectionsUtils.merge(t, null);
        org.junit.jupiter.api.Assertions.assertEquals(t, result);
        org.junit.jupiter.api.Assertions.assertNotSame(t, result);
    }

    @org.junit.jupiter.api.Test
    public void testMergeMapOverlapJunit5() {
        Map<String, String> t = new HashMap<>();
        t.put("key", "value1");
        Map<String, String> s = new HashMap<>();
        s.put("key", "value2");
        Map<String, String> result = CollectionsUtils.merge(t, s);
        org.junit.jupiter.api.Assertions.assertEquals("value1", result.get("key"));
    }

    @org.junit.jupiter.api.Test
    public void testMergeListFirstNullJunit5() {
        List<String> s = Arrays.asList("A", "B");
        List<String> result = CollectionsUtils.merge(null, s);
        org.junit.jupiter.api.Assertions.assertEquals(s, result);
    }

    @org.junit.jupiter.api.Test
    public void testMergeListSecondNullJunit5() {
        List<String> t = Arrays.asList("A", "B");
        List<String> result = CollectionsUtils.merge(t, null);
        org.junit.jupiter.api.Assertions.assertEquals(t, result);
        org.junit.jupiter.api.Assertions.assertNotSame(t, result);
    }


    @org.junit.jupiter.api.Test
    public void testPartitionUnique() {
        java.util.List<Integer> list = java.util.Arrays.asList(1, 2, 3, 4, 5);
        java.util.List<java.util.List<Integer>> partitioned = CollectionsUtils.partition(list, 2);
        org.junit.jupiter.api.Assertions.assertEquals(3, partitioned.size());
    }

    @org.junit.jupiter.api.Test
    public void testMergeMapBothNull() {
        org.junit.jupiter.api.Assertions.assertNull(CollectionsUtils.merge((Map<String, Object>) null, null));
    }

    @org.junit.jupiter.api.Test
    public void testMergeListBothNull() {
        org.junit.jupiter.api.Assertions.assertNull(CollectionsUtils.merge((List<Object>) null, null));
    }

    @org.junit.jupiter.api.Test
    public void testMergeListNormalJunit5() {
        List<String> list1 = Arrays.asList("A", "B");
        List<String> list2 = Arrays.asList("C", "D");
        List<String> result = CollectionsUtils.merge(list1, list2);
        org.junit.jupiter.api.Assertions.assertEquals(4, result.size());
        org.junit.jupiter.api.Assertions.assertEquals(Arrays.asList("A", "B", "C", "D"), result);
    }

    @org.junit.jupiter.api.Test
    public void testMergeMapNormalJunit5() {
        Map<String, String> map1 = new HashMap<>();
        map1.put("k1", "v1");
        Map<String, String> map2 = new HashMap<>();
        map2.put("k2", "v2");
        Map<String, String> result = CollectionsUtils.merge(map1, map2);
        org.junit.jupiter.api.Assertions.assertEquals(2, result.size());
        org.junit.jupiter.api.Assertions.assertEquals("v1", result.get("k1"));
        org.junit.jupiter.api.Assertions.assertEquals("v2", result.get("k2"));
    }

    @org.junit.jupiter.api.Test
    public void testPartitionNegativeIndexJunit5() {
        org.junit.jupiter.api.Assertions.assertThrows(IndexOutOfBoundsException.class, () -> {
            CollectionsUtils.partition(Arrays.asList("A", "B", "C"), 2).get(-1);
        });
    }

}

