package ai.open.right.workflow.flow.mapcombine;

import org.junit.Assert;
import org.junit.Test;

public class MapCombineConfigTest {

    @Test
    public void testValid() {
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        mapCombineConfig.setCombine(new Combine());
        mapCombineConfig.setMapping(new Mapping());
        mapCombineConfig.getCombine().setDynamic("Combine");
        mapCombineConfig.getMapping().setDynamic("Map");
        mapCombineConfig.getMapping().setSplit("Split");
        mapCombineConfig.setTimeout4Llm(1000);
        Assert.assertTrue(mapCombineConfig.isValid());
        Assert.assertEquals(Integer.valueOf(1000),mapCombineConfig.getTimeout4Llm());
    }

    @Test
    public void testInit() {
        MapCombineConfig config = new MapCombineConfig();
        config.setMapping(new Mapping());
        config.setCombine(new Combine());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER1").getMapping().getNotifier());
        Assert.assertEquals("NOTIFIER1", config.init("NOTIFIER2").getCombine().getNotifier());
    }

    @Test
    public void testMerge() throws Exception {
        MapCombineConfig target = new MapCombineConfig();
        MapCombineConfig source = new MapCombineConfig();
        MapCombineConfig result = target.merge(null);
        Assert.assertSame(target, result);
        Assert.assertNull(target.getTimeout4Llm());
        Assert.assertNull(target.getCombine());
        Assert.assertNull(target.getMapping());
        source.setTimeout4Llm(5000);
        Combine sourceCombine = new Combine();
        sourceCombine.setBatch(10);
        source.setCombine(sourceCombine);
        Mapping sourceMapping = new Mapping();
        sourceMapping.setSplit("sourceSplit");
        source.setMapping(sourceMapping);
        result = target.merge(source);
        Assert.assertEquals(Integer.valueOf(5000), target.getTimeout4Llm());
        Assert.assertSame(sourceCombine, target.getCombine());
        Assert.assertSame(sourceMapping, target.getMapping());
        MapCombineConfig target2 = new MapCombineConfig();
        target2.setTimeout4Llm(3000);
        Combine target2Combine = new Combine();
        target2Combine.setBatch(5);
        target2.setCombine(target2Combine);
        Mapping target2Mapping = new Mapping();
        target2Mapping.setSplit("targetSplit");
        target2.setMapping(target2Mapping);
        result = target2.merge(source);
        Assert.assertEquals(Integer.valueOf(3000), target2.getTimeout4Llm());
        Assert.assertSame(target2Combine, target2.getCombine());
        Assert.assertSame(target2Mapping, target2.getMapping());
    }
}
