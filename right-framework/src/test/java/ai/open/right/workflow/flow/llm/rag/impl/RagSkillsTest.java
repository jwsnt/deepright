package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.AllowedConfig;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.rag.skills.RagSkills;
import ai.open.right.workflow.flow.llm.rag.skills.RagSkillsConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.skill.SkillMetadata;
import ai.open.right.workflow.skill.Skills;
import ai.open.right.workflow.skill.SkillsFetcher;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;

public class RagSkillsTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagSkills.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagSkills.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        SkillsFetcher skillMetadataFetcher = EasyMock.createMock(SkillsFetcher.class);
        SkillMetadata skillMetadata1 = SkillMetadata.builder()
                .description("A")
                .path("B")
                .name("C")
                .build();
        SkillMetadata skillMetadata2 = SkillMetadata.builder()
                .description("D")
                .path("E")
                .name("F")
                .build();
        EasyMock.expect(skillMetadataFetcher.fetchSkills(EasyMock.anyObject(WorkflowTask.class), EasyMock.anyObject(AllowedConfig.class))).andReturn(
                Skills.builder()
                        .skills(ImmutableMap.of("A", skillMetadata1, "B", skillMetadata2)).usage("USAGE").build()).anyTimes();
        EasyMock.replay(skillMetadataFetcher);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagSkills ragSkills = new RagSkills();
        ragSkills.setExpire(10000);
        ragSkills.setRepeat(10);
        ragSkills.init();
        ragSkills.setSkillFetcher(skillMetadataFetcher);
        ragSkills.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertTrue(ragData.getPrompt().contains("UNKNOWN \n" +
                "```SKILLS_USAGE\n" +
                "USAGE\n" +
                "```\n" +
                "----------\n" +
                "description: \"A\"\n" +
                "name: \"C\"\n" +
                "----------\n" +
                "description: \"D\"\n" +
                "name: \"F\"\n" +
                "----------\n"));
        EasyMock.verify(skillMetadataFetcher);
    }

    @Test
    public void testWithNoneSkill() throws Exception {
        SkillsFetcher skillMetadataFetcher = EasyMock.createMock(SkillsFetcher.class);
        EasyMock.expect(skillMetadataFetcher.fetchSkills(EasyMock.anyObject(WorkflowTask.class), EasyMock.anyObject(AllowedConfig.class))).
                andReturn(Skills.builder().build()).anyTimes();
        EasyMock.replay(skillMetadataFetcher);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagSkills ragSkills = new RagSkills();
        ragSkills.setExpire(10000);
        ragSkills.setRepeat(10);
        ragSkills.init();
        ragSkills.setSkillFetcher(skillMetadataFetcher);
        ragSkills.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertTrue(ragData.getPrompt().contains("UNKNOWN"));
        EasyMock.verify(skillMetadataFetcher);
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .build();
        RagSkills ragSkills = new RagSkills();
        ragSkills.setExpire(10000);
        ragSkills.init();
        ragSkills.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, ragSkills.rag(ragConfig, ragData));
    }

    @Test
    public void testException() throws Exception {
        SkillsFetcher skillMetadataFetcher = EasyMock.createMock(SkillsFetcher.class);
        EasyMock.expect(skillMetadataFetcher.fetchSkills(EasyMock.anyObject(WorkflowTask.class), EasyMock.anyObject(AllowedConfig.class))).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(skillMetadataFetcher);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagSkills ragSkills = new RagSkills();
        ragSkills.setExpire(10000);
        ragSkills.setRepeat(10);
        ragSkills.init();
        ragSkills.setSkillFetcher(skillMetadataFetcher);
        ragSkills.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertEquals(ragData.getPrompt(), "UNKNOWN #key");
        EasyMock.verify(skillMetadataFetcher);
    }

    @Test
    public void testInit() throws Exception {
        SkillsFetcher skillMetadataFetcher = EasyMock.createMock(SkillsFetcher.class);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagSkills.InitConfig service = new RagSkills.InitConfig();
        service.setSkillFetcher(skillMetadataFetcher);
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        service.setExpire(10089);
        service.setRepeat(10);
        EasyMock.replay(skillMetadataFetcher);
        RagSkills empty = service.ragSkills();
        Assert.assertEquals(skillMetadataFetcher, empty.getSkillFetcher());
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(Integer.valueOf(10089), empty.getExpire());
        Assert.assertEquals(Integer.valueOf(10), empty.getRepeat());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
        EasyMock.verify(skillMetadataFetcher);
        empty.init();
        Assert.assertNotNull(empty.getSkillsCache());
    }

    @Test
    public void testWithAllowedConfig() throws Exception {
        SkillsFetcher skillMetadataFetcher = EasyMock.createMock(SkillsFetcher.class);
        SkillMetadata skillMetadata1 = SkillMetadata.builder()
                .description("A")
                .path("B")
                .name("C")
                .build();
        SkillMetadata skillMetadata2 = SkillMetadata.builder()
                .description("D")
                .path("E")
                .name("F")
                .build();
        RagSkillsConfig ragSkillsConfig = new RagSkillsConfig();
        ragSkillsConfig.addWhite("C");
        EasyMock.expect(skillMetadataFetcher.fetchSkills(EasyMock.anyObject(WorkflowTask.class), EasyMock.anyObject(AllowedConfig.class))).andReturn(
                Skills.builder()
                        .skills(ImmutableMap.of("A", skillMetadata1, "B", skillMetadata2)).usage("USAGE").build()).anyTimes();
        EasyMock.replay(skillMetadataFetcher);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagSkillsConfig(ragSkillsConfig);
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagSkills ragSkills = new RagSkills();
        ragSkills.setExpire(10000);
        ragSkills.setRepeat(10);
        ragSkills.init();
        ragSkills.setSkillFetcher(skillMetadataFetcher);
        ragSkills.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertTrue(ragData.getPrompt().contains("UNKNOWN \n" +
                "```SKILLS_USAGE\n" +
                "USAGE\n" +
                "```\n" +
                "----------\n" +
                "description: \"A\"\n" +
                "name: \"C\"\n" +
                "----------"));
        EasyMock.verify(skillMetadataFetcher);
    }

    /**
     * 覆盖 RagSkills 83-85 行：buildMetadata 当 skillMetadata 为空时 return ""。
     */
    @Test
    public void testBuildMetadataEmpty() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("P")
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        RagSkillsForTest ragSkillsForTest = new RagSkillsForTest();
        String result = ragSkillsForTest.callBuildMetadata(ragConfig, ragData, Collections.emptyList());
        Assert.assertEquals("", result);
    }

    /**
     * 覆盖 RagSkills 104-106 行：buildUsage 当 skills.getUsage() 为空时 return ""。
     */
    @Test
    public void testBuildUsageEmpty() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("P")
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        Skills skillsEmptyUsage = Skills.builder().skills(ImmutableMap.of()).build();
        RagSkillsForTest ragSkillsForTest = new RagSkillsForTest();
        String result = ragSkillsForTest.callBuildUsage(ragConfig, ragData, skillsEmptyUsage);
        Assert.assertEquals("", result);
    }

    /**
     * 覆盖 allowedSkill：ragSkillsConfig == null 时返回 true。
     */
    @Test
    public void testAllowedSkillWhenRagSkillsConfigNull() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagSkillsConfig(null);
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("P")
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        SkillMetadata skill = SkillMetadata.builder().name("any").build();
        RagSkillsForTest ragSkillsForTest = new RagSkillsForTest();
        Assert.assertTrue(ragSkillsForTest.callAllowedSkill(ragConfig, ragData, skill));
    }

    /**
     * 覆盖 allowedSkill：ragSkillsConfig != null 且 allowed(skill.getName()) 为 true。
     */
    @Test
    public void testAllowedSkillWhenAllowedReturnsTrue() throws Exception {
        RagSkillsConfig ragSkillsConfig = new RagSkillsConfig();
        ragSkillsConfig.addWhite("skillA");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagSkillsConfig(ragSkillsConfig);
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("P")
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        SkillMetadata skill = SkillMetadata.builder().name("skillA").build();
        RagSkillsForTest ragSkillsForTest = new RagSkillsForTest();
        Assert.assertTrue(ragSkillsForTest.callAllowedSkill(ragConfig, ragData, skill));
    }

    /**
     * 覆盖 allowedSkill：ragSkillsConfig != null 且 allowed(skill.getName()) 为 false。
     */
    @Test
    public void testAllowedSkillWhenAllowedReturnsFalse() throws Exception {
        RagSkillsConfig ragSkillsConfig = new RagSkillsConfig();
        ragSkillsConfig.addWhite("onlyThis");
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagSkillsConfig(ragSkillsConfig);
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("P")
                .query(ObjectBuilder.buildLLMQuery())
                .build();
        SkillMetadata skill = SkillMetadata.builder().name("otherSkill").build();
        RagSkillsForTest ragSkillsForTest = new RagSkillsForTest();
        Assert.assertFalse(ragSkillsForTest.callAllowedSkill(ragConfig, ragData, skill));
    }

    private static class RagSkillsForTest extends RagSkills {
        String callBuildMetadata(RagConfig c, RagData d, java.util.Collection<SkillMetadata> skillMetadata) throws Exception {
            return buildMetadata(c, d, skillMetadata);
        }

        String callBuildUsage(RagConfig c, RagData d, Skills s) throws Exception {
            return buildUsage(c, d, s);
        }

        Boolean callAllowedSkill(RagConfig c, RagData d, SkillMetadata skill) throws Exception {
            return allowedSkill(c, d, skill);
        }
    }
}
