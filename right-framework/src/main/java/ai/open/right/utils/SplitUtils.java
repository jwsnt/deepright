package ai.open.right.utils;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

import java.util.Arrays;

@Slf4j
public class SplitUtils {

    public static final String SPLIT_SLASH = "/";

    public static final String SPLIT_AT = "@";

    public static String[] splitWithSymbol(String symbol, String scene, String biz) {
        if (!scene.contains(symbol)) {
            String[] part = new String[]{biz, scene};
            if (log.isDebugEnabled()) {
                log.debug("Split part={}", Arrays.toString(part));
            }
            return part;
        }
        return SplitUtils.splitWithSymbol(symbol, scene);
    }

    public static String[] splitWithSymbol(String symbol, String scene) {
        String[] part = scene.split(symbol);
        Assert.isTrue(part.length >= 2, "Biz or workflow is invalid: " + scene);
        if (log.isDebugEnabled()) {
            log.debug("Split part={}", Arrays.toString(part));
        }
        return part;
    }

    // 解析biz@workflow格式的Scene字符串，如果无法解析biz则使用传入值
    public static String[] split(String scene, String biz) {
        return SplitUtils.splitWithSymbol(SplitUtils.SPLIT_AT, scene, biz);
    }

    public static String[] split(String scene) {
        return SplitUtils.splitWithSymbol(SplitUtils.SPLIT_AT, scene);
    }

    public static String join(String delimiter, String scene, String biz) {
        String[] part = SplitUtils.split(scene, biz);
        return part[0] + delimiter + part[1];
    }

    public static String join(String scene, String biz) {
        return SplitUtils.join(SplitUtils.SPLIT_AT, scene, biz);
    }

    public static String join(WorkflowTask workTask) {
        return SplitUtils.join(SplitUtils.SPLIT_AT, workTask.getWorkflow(), workTask.getBiz());
    }

    public static String join(Segment segment) {
        return SplitUtils.join(SplitUtils.SPLIT_AT, segment.getWorkflow(), segment.getBiz());
    }

    public static String join(String[] parts) {
        return SplitUtils.join(SplitUtils.SPLIT_AT, parts[1], parts[0]);
    }

    // 获取实际BIZ, scene可能为biz@workflow
    public static String biz(String scene, String biz) {
        return SplitUtils.split(scene, biz)[0];
    }

    public static Boolean equals(String scene, String biz, String parts) {
        return StringUtils.endsWithIgnoreCase(SplitUtils.join(scene, biz), parts);
    }

    public static Boolean equals(WorkflowTask workTask, String parts) {
        return StringUtils.endsWithIgnoreCase(SplitUtils.join(workTask.getWorkflow(), workTask.getBiz()), parts);
    }

    public static Boolean equals(Segment segment, String parts) {
        return StringUtils.endsWithIgnoreCase(SplitUtils.join(segment.getWorkflow(), segment.getBiz()), parts);
    }
}
