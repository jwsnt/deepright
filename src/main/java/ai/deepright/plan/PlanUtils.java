package ai.deepright.plan;

import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureField;
import ai.deepright.router.RouterDevice;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.text.similarity.JaroWinklerSimilarity;
import org.apache.commons.text.similarity.LevenshteinDistance;

import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.regex.PatternSyntaxException;

@Slf4j
public class PlanUtils {

    public static final int MAX_FUZZY_LINE_LENGTH = 512;

    public static final JaroWinklerSimilarity JWS = new JaroWinklerSimilarity();

    // 是否主动关闭规划
    public static final String KEY_PLAN_DISABLE = "plan_disable";

    public static final String KEY_PLAN_TIME = "_time";

    public static final String KEY_PLAN = "plan";

    public static final Double DISTANCE = 0.7;

    public static String fetchPlan(WorkflowTask workTask) throws Exception {
        // 如果是Task任务则添加后缀
        String key = PlanUtils.KEY_PLAN + StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_UPSTREAM), RouterDevice.key(workTask));
        return workTask.getUserContext().getMetadata(key, String.class);
    }

    public static Long fetchTime(WorkflowTask workTask) throws Exception {
        // 如果是Task任务则添加后缀
        String key = PlanUtils.KEY_PLAN + StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_UPSTREAM), RouterDevice.key(workTask));
        Long time = workTask.getUserContext().getMetadata(key + PlanUtils.KEY_PLAN_TIME, Long.class);
        workTask.getUserContext().putMetadata(key + PlanUtils.KEY_PLAN_TIME, time = time != null ? time : System.currentTimeMillis());
        return time;
    }

    // 为当前Router存储规划
    public static String storePlan(WorkflowTask workTask, String plan) throws Exception {
        // 追加当前Router在跨Task时不冲突
        String key = PlanUtils.KEY_PLAN + RouterDevice.key(workTask);
        workTask.getUserContext().putMetadata(key, PlanEncode.replace(plan));
        workTask.getUserContext().delMetadata(key + PlanUtils.KEY_PLAN_TIME, Long.class);
        return key;
    }

    public static String cleanPlan(WorkflowTask workTask, String key) throws Exception {
        String disable = PlanUtils.KEY_PLAN_DISABLE + StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_UPSTREAM), key);
        String store = PlanUtils.KEY_PLAN + StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_UPSTREAM), key);
        workTask.getUserContext().getMetadata().remove(disable + PlanUtils.KEY_PLAN_TIME);
        workTask.getUserContext().getMetadata().remove(disable);
        return String.class.cast(workTask.getUserContext().getMetadata().remove(store));
    }

    // 是否需要规划
    public static Boolean shouldPlan(WorkflowTask workTask) throws Exception {
        // 标记
        String key = PlanUtils.KEY_PLAN_DISABLE + StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_UPSTREAM), RouterDevice.key(workTask));
        if (MapUtils.getBoolean(workTask.getUserContext().getMetadata(), key, false)) {
            return false;
        }
        // 未标记
        if (ComplexityUtils.result(workTask).is(ComplexityMode.FAST_REPLY)) {
            // 复杂度不足，标记
            PlanUtils.disablePlan(workTask);
            return false;
        } else {
            return true;
        }
    }

    public static void disablePlan(WorkflowTask workTask) throws Exception {
        String key = PlanUtils.KEY_PLAN_DISABLE + StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_ROUTER_UPSTREAM), RouterDevice.key(workTask));
        workTask.getUserContext().putMetadata(key, true);
    }

    // 通过PlanPattern替换Content中的所有匹配内容
    public static String replace(WorkflowTask workTask, String content, List<PlanPattern> pattern) throws Exception {
        String current = content;
        if (!CollectionUtils.isEmpty(pattern)) {
            for (PlanPattern each : pattern) {
                String replaceString = PlanEncode.replace(each.getReplacement());
                String patternString = PlanEncode.replace(each.getPattern());
                // 精确替换
                String replace = current.replace(patternString, replaceString);
                // 区分大小写变化
                if (StringUtils.equals(replace, current)) {
                    // 如果替换失败（无变化）则正则替换（不转义）
                    try {
                        replace = current.replaceAll(patternString, Matcher.quoteReplacement(replaceString));
                    } catch (PatternSyntaxException e) {
                        // 正则特殊字符会抛PatternSyntaxException
                        if (log.isDebugEnabled()) {
                            log.debug(e.getMessage(), e);
                        }
                    }
                }
                // 区分大小写变化
                if (StringUtils.equals(replace, current)) {
                    // 如果替换失败（无变化）则正则替换（转义）
                    replace = current.replaceAll(Pattern.quote(patternString), Matcher.quoteReplacement(replaceString));
                }
                // 区分大小写变化
                if (StringUtils.equals(replace, current)) {
                    // 如果替换失败（无变化）则按行模糊匹配，避免整块内容被单行文本覆盖
                    replace = PlanUtils.replaceSimilarLine(current, patternString, replaceString);
                }
                // 区分大小写变化
                if (StringUtils.equals(replace, current)) {
                    // 如果替换失败（无变化）则直接追加并提示日志
                    replace = current + replaceString;
                    if (log.isDebugEnabled()) {
                        log.debug("The plan update failed, please ensure the replacement text is an `exact match`. pattern={}, replace={}, content={}", patternString, replaceString, content);
                    }
                }
                current = replace;
            }
        }
        return current;
    }

    protected static String replaceSimilarLine(String content, String patternString, String replaceString) {
        if (!PlanUtils.isSingleLine(patternString) || !PlanUtils.isSingleLine(replaceString) || StringUtils.isBlank(patternString)) {
            return content;
        }
        if (patternString.length() > PlanUtils.MAX_FUZZY_LINE_LENGTH || replaceString.length() > PlanUtils.MAX_FUZZY_LINE_LENGTH) {
            return content;
        }
        int threshold = Math.max(1, (int) Math.ceil(patternString.length() * (1 - PlanUtils.DISTANCE)));
        LevenshteinDistance distance = new LevenshteinDistance(threshold);
        int bestLineStart = -1;
        int bestLineEnd = -1;
        int bestDistance = Integer.MAX_VALUE;
        double bestSimilarity = -1D;
        boolean ambiguous = false;
        int index = 0;
        while (index < content.length()) {
            int lineStart = index;
            int lineEnd = index;
            while (lineEnd < content.length() && content.charAt(lineEnd) != '\n' && content.charAt(lineEnd) != '\r') {
                lineEnd++;
            }
            int separatorEnd = lineEnd;
            if (separatorEnd < content.length()) {
                if (content.charAt(separatorEnd) == '\r' && separatorEnd + 1 < content.length() && content.charAt(separatorEnd + 1) == '\n') {
                    separatorEnd += 2;
                } else {
                    separatorEnd++;
                }
            }
            String line = content.substring(lineStart, lineEnd);
            if (line.length() <= PlanUtils.MAX_FUZZY_LINE_LENGTH && Math.abs(line.length() - patternString.length()) <= threshold) {
                Integer lineDistance = distance.apply(line, patternString);
                Double lineSimilarity = PlanUtils.JWS.apply(line, patternString);
                if ((lineDistance != null && lineDistance >= 0) || lineSimilarity >= PlanUtils.DISTANCE) {
                    if (lineDistance == null || lineDistance < 0) {
                        lineDistance = threshold + 1;
                    }
                    if (lineDistance < bestDistance || lineDistance == bestDistance && lineSimilarity > bestSimilarity) {
                        bestLineStart = lineStart;
                        bestLineEnd = lineEnd;
                        bestDistance = lineDistance;
                        bestSimilarity = lineSimilarity;
                        ambiguous = false;
                    } else if (lineDistance == bestDistance && Double.compare(lineSimilarity, bestSimilarity) == 0) {
                        ambiguous = true;
                    }
                }
            }
            index = separatorEnd;
        }
        if (bestLineStart < 0 || ambiguous) {
            return content;
        }
        if (log.isInfoEnabled()) {
            log.info("The plan has been fuzzy-matched by line. pattern={}, replacement={}", patternString, replaceString);
        }
        return content.substring(0, bestLineStart) + replaceString + content.substring(bestLineEnd);
    }

    protected static Boolean isSingleLine(String content) {
        return !StringUtils.containsAny(content, '\r', '\n');
    }
}
