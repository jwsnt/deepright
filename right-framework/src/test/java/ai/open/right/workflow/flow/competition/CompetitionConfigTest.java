package ai.open.right.workflow.flow.competition;

import ai.open.right.utils.JsonUtils;
import org.junit.Assert;
import org.junit.Test;

public class CompetitionConfigTest {

    @Test
    public void test() {
        CompetitionConfig competitionConfig = new CompetitionConfig();
        competitionConfig.setTimeout(1000);
        Assert.assertEquals(Integer.valueOf(1000), competitionConfig.getTimeout());
        Assert.assertEquals(Integer.valueOf(1000), competitionConfig.getTimeout(2000));
    }

    @Test
    public void test2() {
        CompetitionConfig competitionConfig = new CompetitionConfig();
        Assert.assertEquals(Integer.valueOf(2000), competitionConfig.getTimeout(2000));
    }

    @Test
    public void testMerge_AllNullsUseSource() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        source.setStopOnFailed(Boolean.TRUE);
        source.setTimeout(3000);
        source.setDynamic("fallback");
        target.merge(source);
        Assert.assertEquals(Boolean.TRUE, target.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(3000), target.getTimeout());
        Assert.assertEquals("fallback", target.getDynamic());
    }

    @Test
    public void testMerge_TargetValuesPreserved() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        target.setStopOnFailed(Boolean.FALSE);
        target.setTimeout(4000);
        target.setDynamic("targetDyn");
        source.setStopOnFailed(Boolean.TRUE);
        source.setTimeout(1000);
        source.setDynamic("sourceDyn");
        target.merge(source);
        Assert.assertEquals(Boolean.FALSE, target.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(4000), target.getTimeout());
        Assert.assertEquals("targetDyn", target.getDynamic());
    }

    @Test
    public void testMerge_BlankDynamicUsesSource() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        target.setDynamic("   ");
        source.setDynamic("sourceDyn");
        target.merge(source);
        Assert.assertEquals("sourceDyn", target.getDynamic());
    }

    @Test
    public void testMerge_NullSourceNoChange() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        target.setStopOnFailed(Boolean.TRUE);
        target.setTimeout(5000);
        target.setDynamic("dyn");
        target.merge(null);
        Assert.assertEquals(Boolean.TRUE, target.getStopOnFailed());
        Assert.assertEquals(Integer.valueOf(5000), target.getTimeout());
        Assert.assertEquals("dyn", target.getDynamic());
    }

    @Test
    public void testMerge_ConditionConfigs_UseSourceWhenTargetNull() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        java.util.List<ConditionConfig> sourceList = new java.util.ArrayList<>();
        sourceList.add(new ConditionConfig());
        sourceList.add(new ConditionConfig());
        source.setConditionConfigs(sourceList);
        target.merge(source);
        Assert.assertTrue(target.hasConditions());
        Assert.assertSame(sourceList, target.getConditionConfigs());
        Assert.assertEquals(2, target.getConditionConfigs().size());
    }

    @Test
    public void testMerge_ConditionConfigs_PreserveTargetWhenPresent() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        java.util.List<ConditionConfig> targetList = new java.util.ArrayList<>();
        targetList.add(new ConditionConfig());
        target.setConditionConfigs(targetList);
        java.util.List<ConditionConfig> sourceList = new java.util.ArrayList<>();
        sourceList.add(new ConditionConfig());
        sourceList.add(new ConditionConfig());
        source.setConditionConfigs(sourceList);
        target.merge(source);
        Assert.assertEquals("[{},{},{}]", JsonUtils.write(target.getConditionConfigs()));
        Assert.assertEquals(3, target.getConditionConfigs().size());
    }

    @Test
    public void testMerge_SourceBlankDynamicDoesNotOverride() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        target.setDynamic("targetDyn");
        source.setDynamic("   ");
        target.merge(source);
        Assert.assertEquals("targetDyn", target.getDynamic());
    }

    @Test
    public void testMerge_BothNullsRemainNullOrDefaults() throws Exception {
        CompetitionConfig target = new CompetitionConfig();
        CompetitionConfig source = new CompetitionConfig();
        target.merge(source);
        Assert.assertNull(target.getTimeout());
        Assert.assertFalse(target.getStopOnFailed());
        Assert.assertFalse(target.hasTarget());
        Assert.assertFalse(target.hasConditions());
    }
}
