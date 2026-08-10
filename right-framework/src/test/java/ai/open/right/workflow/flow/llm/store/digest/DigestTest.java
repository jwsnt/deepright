package ai.open.right.workflow.flow.llm.store.digest;

import ai.open.right.workflow.flow.llm.store.digest.Digest;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class DigestTest {

    @Test
    public void testMergeEmpty() {
        Map<String, Object> maps1 = new HashMap<String, Object>();
        Map<String, Object> maps2 = new HashMap<String, Object>();
        List<String> keys = new ArrayList<String>();
        Digest digest = new Digest(maps1, keys);
        digest.merge(maps2);
        Assert.assertEquals(0, digest.getDigest().size());
        Assert.assertEquals(keys, digest.getKeys());
    }

    @Test
    public void testMerge1() {
        Map<String, Object> maps1 = new HashMap<String, Object>();
        maps1.put("DiscordConfigTest", "B");
        Map<String, Object> maps2 = new HashMap<String, Object>();
        maps2.put("C", "D");
        List<String> keys = new ArrayList<String>();
        keys.add("DiscordConfigTest");
        keys.add("B");
        Digest digest = new Digest(maps1, keys);
        digest.merge(maps2);
        Assert.assertEquals(2, digest.getDigest().size());
    }

    @Test
    public void testMerge3() {
        Map<String, Object> maps1 = new HashMap<String, Object>();
        maps1.put("DiscordConfigTest", "B");
        Map<String, Object> maps2 = new HashMap<String, Object>();
        maps2.put("C", "D");
        List<String> keys = new ArrayList<String>();
        keys.add("DiscordConfigTest");
        Digest digest = new Digest(maps1, keys);
        digest.merge(maps2);
        Assert.assertEquals(1, digest.getDigest().size());
        Assert.assertTrue(digest.getDigest().containsKey("DiscordConfigTest"));
    }

    @Test
    public void testMerge4() {
        Map<String, Object> maps1 = new HashMap<String, Object>();
        maps1.put("DiscordConfigTest", "B");
        Map<String, Object> maps2 = new HashMap<String, Object>();
        maps2.put("DiscordConfigTest", "C");
        List<String> keys = new ArrayList<String>();
        keys.add("DiscordConfigTest");
        Digest digest = new Digest(maps1, keys);
        digest.merge(maps2);
        Assert.assertEquals("B", digest.getDigest().get("DiscordConfigTest"));
    }

    @Test
    public void testMergeNoKeys() {
        Map<String, Object> maps1 = new HashMap<String, Object>();
        maps1.put("DiscordConfigTest", "B");
        Map<String, Object> maps2 = new HashMap<String, Object>();
        maps2.put("DiscordConfigTest", "C");
        List<String> keys = new ArrayList<String>();
        Digest digest = new Digest(maps1, keys);
        digest.merge(maps2);
        Assert.assertEquals("B", digest.getDigest().get("DiscordConfigTest"));
    }
}
