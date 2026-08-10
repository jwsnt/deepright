package ai.open.right.utils;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import org.junit.Assert;
import org.junit.Test;

/**
 * Unit tests for SplitUtils using JUnit 4.
 */
public class SplitUtilsTest {

    @Test
    public void testBasicSplitWithDefaultBiz() {
        String[] pair = SplitUtils.split("workflow", "biz");
        Assert.assertEquals("biz", pair[0]);
        Assert.assertEquals("workflow", pair[1]);
    }

    @Test
    public void testBasicSplitWithProvidedBiz() {
        String[] pair = SplitUtils.split("biz2@workflow", "biz");
        Assert.assertEquals("biz2", pair[0]);
        Assert.assertEquals("workflow", pair[1]);
    }

    @Test
    public void testMultipleAt() {
        // "biz@workflow@extra".split("@") returns ["biz", "workflow", "extra"]
        String[] pair = SplitUtils.split("biz@workflow@extra", "default");
        Assert.assertTrue(pair.length >= 2);
        Assert.assertEquals("biz", pair[0]);
        Assert.assertEquals("workflow", pair[1]);
        Assert.assertEquals("extra", pair[2]);
    }

    @Test
    public void testStartWithAt() {
        // "@workflow".split("@") returns ["", "workflow"]
        String[] pair = SplitUtils.split("@workflow", "default");
        Assert.assertEquals("", pair[0]);
        Assert.assertEquals("workflow", pair[1]);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testEndWithAt() {
        // "biz@".split("@") returns ["biz"], which has length 1 and triggers IllegalArgumentException
        SplitUtils.split("biz@", "default");
    }

    @Test
    public void testEmptyScene() {
        // Empty scene does not contain "@", so it returns [biz, scene]
        String[] pair = SplitUtils.split("", "default");
        Assert.assertEquals("default", pair[0]);
        Assert.assertEquals("", pair[1]);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testInvalidSplit() {
        // split(String scene) requires the symbol to be present and result in at least 2 parts
        SplitUtils.split("no-at");
    }

    @Test
    public void testJoinWithSymbol() {
        // join splits the scene and then joins with the provided delimiter
        String result = SplitUtils.join("-", "biz@workflow", "default");
        Assert.assertEquals("biz-workflow", result);
    }

    @Test
    public void testBizExtraction() {
        Assert.assertEquals("biz", SplitUtils.biz("biz@workflow", "default"));
        Assert.assertEquals("default", SplitUtils.biz("workflow", "default"));
    }

    @org.junit.jupiter.api.Test
    public void testSplitMultipleSymbols() {
        String[] parts = SplitUtils.split("a@b@c", "default");
        org.junit.jupiter.api.Assertions.assertEquals(3, parts.length);
        org.junit.jupiter.api.Assertions.assertEquals("a", parts[0]);
    }

    @org.junit.jupiter.api.Test
    public void testSplitNoSymbolWithDefault() {
        String[] parts = SplitUtils.split("nosymbol", "def");
        org.junit.jupiter.api.Assertions.assertEquals("def", parts[0]);
        org.junit.jupiter.api.Assertions.assertEquals("nosymbol", parts[1]);
    }

    @org.junit.jupiter.api.Test
    public void testSplitWithNullScene() {
        // 修正：根据代码逻辑，scene 为 null 时会触发 NullPointerException
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            SplitUtils.split(null, "default");
        });
    }

    @org.junit.jupiter.api.Test
    public void testSplitNullSingleArg() {
        // 修正：单参数 split 传入 null 时，内部调用 scene.split 会抛出 NullPointerException
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            SplitUtils.split(null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testConsecutiveAt() {
        // 边界场景：包含连续的分隔符
        String[] pair = SplitUtils.split("biz@@workflow", "default");
        org.junit.jupiter.api.Assertions.assertEquals(3, pair.length);
        org.junit.jupiter.api.Assertions.assertEquals("biz", pair[0]);
        org.junit.jupiter.api.Assertions.assertEquals("", pair[1]);
        org.junit.jupiter.api.Assertions.assertEquals("workflow", pair[2]);
    }

    @org.junit.jupiter.api.Test
    public void testBizWithNullScene() {
        // 修正：获取 null 场景的业务标识会因内部调用 split 抛出 NullPointerException
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            SplitUtils.biz(null, "default");
        });
    }

    @org.junit.jupiter.api.Test
    public void testSplitWithRegexSpecialChar() {
        // 边界场景：测试正则特殊字符作为分隔符。注意：splitWithSymbol 内部直接使用 String.split(symbol)
        // 如果 symbol 是 "."，它在正则中代表匹配任意字符，会导致非预期分割
        String dotSymbol = ".";
        String scene = "biz.workflow";
        
        // 修正：验证包含特殊字符但未转义时的行为。由于 "." 是正则保留字符，
        // scene.split(".") 会匹配所有字符并返回空数组，从而触发 SplitUtils 中的 Assert.isTrue 抛出 IllegalArgumentException
        org.junit.jupiter.api.Assertions.assertThrows(IllegalArgumentException.class, () -> {
            SplitUtils.splitWithSymbol(dotSymbol, scene, "default");
        });

        // 验证正常字符作为分隔符
        String normalSymbol = "#";
        String[] normalParts = SplitUtils.splitWithSymbol(normalSymbol, "biz#workflow", "default");
        org.junit.jupiter.api.Assertions.assertEquals("biz", normalParts[0]);
        org.junit.jupiter.api.Assertions.assertEquals("workflow", normalParts[1]);
    }


    @org.junit.jupiter.api.Test
    public void testSplitAdditionalUnique() {
        String input = "a,b,c";
        java.util.List<String> result = java.util.Arrays.asList(input.split(","));
        org.junit.jupiter.api.Assertions.assertEquals(3, result.size());
    }

    @org.junit.jupiter.api.Test
    public void testJoinNull() {
        // 修正：Java 字符串拼接 null 不会抛 NPE，而是拼接 "null" 字符串
        String result = SplitUtils.join(null, "a@b", "biz");
        org.junit.jupiter.api.Assertions.assertEquals("anullb", result);
    }

    @org.junit.jupiter.api.Test
    public void testSplitWithCustomSymbol() {
        String[] parts = SplitUtils.splitWithSymbol("#", "a#b");
        org.junit.jupiter.api.Assertions.assertEquals(2, parts.length);
        org.junit.jupiter.api.Assertions.assertEquals("a", parts[0]);
        org.junit.jupiter.api.Assertions.assertEquals("b", parts[1]);
    }

    @Test
    public void testJoinWorkflowTask_defaultValues() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        // ObjectBuilder 设置 workflow=UNKNOWN, biz=UNKNOWN
        String result = SplitUtils.join(task);
        Assert.assertEquals("UNKNOWN@UNKNOWN", result);
    }

    @Test
    public void testJoinWorkflowTask_customValues() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("myWorkflow");
        task.setBiz("myBiz");
        String result = SplitUtils.join(task);
        Assert.assertEquals("myBiz@myWorkflow", result);
    }

    @Test
    public void testJoinWorkflowTask_workflowContainsAt() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("biz2@wf");
        task.setBiz("biz1");
        // workflow="biz2@wf" 包含 @，split 会解析为 biz2 和 wf
        String result = SplitUtils.join(task);
        Assert.assertEquals("biz2@wf", result);
    }

    @Test
    public void testJoinWorkflowTask_emptyWorkflowAndBiz() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("");
        task.setBiz("");
        String result = SplitUtils.join(task);
        Assert.assertEquals("@", result);
    }

    @Test
    public void testJoinWorkflowTask_equivalentToJoinAtWorkflowBiz() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("sceneNoAt");
        task.setBiz("fallbackBiz");
        String fromTask = SplitUtils.join(task);
        String fromArgs = SplitUtils.join(SplitUtils.SPLIT_AT, task.getWorkflow(), task.getBiz());
        Assert.assertEquals(fromArgs, fromTask);
        Assert.assertEquals("fallbackBiz@sceneNoAt", fromTask);
    }

    @Test
    public void testJoinWorkflowTask_multipleAtKeepsFirstTwoSegmentsOnly() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("a@b@c");
        task.setBiz("ignoredWhenSceneHasAt");
        Assert.assertEquals("a@b", SplitUtils.join(task));
    }

    @org.junit.jupiter.api.Test
    public void testJoinWorkflowTask_nullTask() {
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class,
                () -> SplitUtils.join((ai.open.right.workflow.flow.WorkflowTask) null));
    }

    @Test
    public void testJoinSegment_customValues() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("myWorkflow");
        task.setBiz("myBiz");

        Segment segment = new SegmentDelegate(task);

        Assert.assertEquals("myBiz@myWorkflow", SplitUtils.join(segment));
    }

    @Test
    public void testJoinSegment_workflowContainsAt() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("biz2@wf");
        task.setBiz("biz1");

        Segment segment = new SegmentDelegate(task);

        Assert.assertEquals("biz2@wf", SplitUtils.join(segment));
    }

    @Test
    public void testJoinSegment_equivalentToJoinAtWorkflowBiz() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("sceneNoAt");
        task.setBiz("fallbackBiz");

        Segment segment = new SegmentDelegate(task);

        String fromSegment = SplitUtils.join(segment);
        String fromArgs = SplitUtils.join(SplitUtils.SPLIT_AT, segment.getWorkflow(), segment.getBiz());
        Assert.assertEquals(fromArgs, fromSegment);
        Assert.assertEquals("fallbackBiz@sceneNoAt", fromSegment);
    }

    @Test
    public void testJoinWorkflowTaskAndSegment_areConsistent() {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("sharedWorkflow");
        task.setBiz("sharedBiz");

        Segment segment = new SegmentDelegate(task);

        Assert.assertEquals(SplitUtils.join(task), SplitUtils.join(segment));
    }

    @org.junit.jupiter.api.Test
    public void testJoinSegment_nullSegment() {
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class,
                () -> SplitUtils.join((Segment) null));
    }

    @Test
    public void testEqualsSceneBiz_trueWhenPartsMatchesJoinIgnoreCase() {
        Assert.assertTrue(SplitUtils.equals("MyWorkflow", "myBiz", "myBiz@MyWorkflow"));
        Assert.assertTrue(SplitUtils.equals("MyWorkflow", "myBiz", "MYBIZ@MYWORKFLOW"));
    }

    @Test
    public void testEqualsSceneBiz_trueWhenSceneContainsAt() {
        // join("biz2@wf", "biz1") => split 得 biz2、wf => "biz2@wf"
        Assert.assertTrue(SplitUtils.equals("biz2@wf", "biz1", "biz2@wf"));
    }

    @Test
    public void testEqualsSceneBiz_falseWhenPartsMismatch() {
        Assert.assertFalse(SplitUtils.equals("w", "b", "b@w@extra"));
        Assert.assertFalse(SplitUtils.equals("workflow", "biz", "wrong"));
    }

    @Test
    public void testEqualsSceneBiz_trueWhenPartsIsSuffix() {
        // join("a", "b") => "b@a"，仅后缀匹配
        Assert.assertTrue(SplitUtils.equals("a", "b", "@a"));
    }

    @Test
    public void testEqualsWorkflowTask_matchesJoinOfWorkflowAndBiz() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("sceneNoAt");
        task.setBiz("fallbackBiz");
        String joined = SplitUtils.join(task);
        Assert.assertTrue(SplitUtils.equals(task, joined));
        Assert.assertTrue(SplitUtils.equals(task, "FALLBACKBIZ@SCENENOAT"));
    }

    @Test
    public void testEqualsWorkflowTask_consistentWithThreeArgEquals() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("wf");
        task.setBiz("biz");
        String parts = "biz@wf";
        Assert.assertEquals(SplitUtils.equals(task.getWorkflow(), task.getBiz(), parts), SplitUtils.equals(task, parts));
    }

    @Test
    public void testEqualsWorkflowTask_falseWhenSuffixMismatch() {
        ai.open.right.workflow.flow.WorkflowTask task = ai.open.right.ObjectBuilder.buildWorkflowTask();
        task.setWorkflow("w");
        task.setBiz("b");
        Assert.assertFalse(SplitUtils.equals(task, "not_a_suffix"));
    }

    @Test
    public void testEqualsSegment_matchesJoinOfWorkflowAndBizIgnoreCase() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        wt.setWorkflow("MyWf");
        wt.setBiz("myBiz");
        Segment segment = new SegmentDelegate(wt);
        String joined = SplitUtils.join(segment.getWorkflow(), segment.getBiz());
        Assert.assertTrue(SplitUtils.equals(segment, joined));
        Assert.assertTrue(SplitUtils.equals(segment, "MYBIZ@MYWF"));
    }

    @Test
    public void testEqualsSegment_consistentWithThreeArgEquals() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        wt.setWorkflow("wf");
        wt.setBiz("biz");
        Segment segment = new SegmentDelegate(wt);
        String parts = "biz@wf";
        Assert.assertEquals(SplitUtils.equals(segment.getWorkflow(), segment.getBiz(), parts), SplitUtils.equals(segment, parts));
    }

    @Test
    public void testEqualsSegment_trueWhenSceneWorkflowContainsAt() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        wt.setWorkflow("biz2@wf");
        wt.setBiz("biz1");
        Segment segment = new SegmentDelegate(wt);
        Assert.assertTrue(SplitUtils.equals(segment, "biz2@wf"));
    }

    @Test
    public void testEqualsSegment_falseWhenSuffixMismatch() {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        wt.setWorkflow("w");
        wt.setBiz("b");
        Segment segment = new SegmentDelegate(wt);
        Assert.assertFalse(SplitUtils.equals(segment, "no_match"));
    }

    @org.junit.jupiter.api.Test
    public void testEqualsSegment_nullSegment_throwsNpe() {
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class,
                () -> SplitUtils.equals((Segment) null, "x"));
    }
}
