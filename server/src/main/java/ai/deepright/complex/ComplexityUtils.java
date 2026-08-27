package ai.deepright.complex;

import ai.deepright.complex.utils.*;
import ai.deepright.feature.FeatureField;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

@Slf4j
public class ComplexityUtils {

    public static final Integer MAX_LENGTH = Integer.valueOf(StringUtils.defaultIfEmpty(System.getenv("COMPLEXITY_MAX_LENGTH"), "5000"));

    // 耗时监控
    public static final Integer THRESHOLD = Integer.valueOf(StringUtils.defaultIfEmpty(System.getenv("COMPLEXITY_THRESHOLD"), "1000"));

    public static final String KEY_COMPLEXITY_UPGRADE = "complexity_upgrade";

    public static final String KEY_COMPLEXITY = "complexity";

    public static ComplexityMode score(String input) throws Exception {
        if (StringUtils.length(input) > ComplexityUtils.MAX_LENGTH) {
            // 超长输入，不做判断
            return ComplexityMode.DEEP_THINKING;
        }
        long startTime = System.currentTimeMillis();
        // --- 维度 1: 结构复杂度 (Structure Score) -> 侧重“任务规划” ---
        // 包含换行缩进、符号嵌套、以及块熵的波动率
        double structureScore = (WhitespaceUtils.score(input) * 0.4)
                + (EnclosureUtils.score(input) * 0.3)
                + (Math.min(BlockDynamicUtils.score(input) * 2.0, 1.0) * 0.3); // 放大块波动率的影响
        // --- 维度 2: 语义密度 (Semantic Density) -> 侧重“深度思考” ---
        // 结合香农熵、LZ压缩比、LSH一致性以及字符集多样性
        double shannonNorm = Math.min(ShannonEntropyUtils.score(input) / 5.0, 1.0); // 归一化香农熵
        double semanticScore = (shannonNorm * 0.3)
                + (CompressionComplexityUtils.score(input) * 0.3) // 压缩比反映信息增量
                + (LSHConsistencyUtils.score(input) * 0.2)        // 局部一致性反映逻辑性
                + (CharsetEntropyUtils.score(input) * 0.2);       // 字符多样性
        // --- 维度 3: 逻辑合法性与噪音过滤 (Logic & Noise Filter) ---
        // HeapsLaw 验证是否符合人类语言规律, TransitionMatrix 验证语法强度
        double logicValidity = (HeapsLawUtils.score(input) * 0.5) + (TransitionMatrixUtils.score(input) * 0.5);
        // 惩罚项：重复率越高、自相关性越异常，则扣分越多
        double penalty = (NGramSelfOverlapUtils.score(input) * 0.8) // 使用 Auto-N 自动计算重复率
                + (AutoCorrelationUtils.score(input) > 0.8 ? 0.2 : 0.0);
        // --- 最终加权总分 ---
        double score = ((structureScore * 0.45) + (semanticScore * 0.45) + (logicValidity * 0.1)) - penalty;
        score = Math.max(0, Math.min(score, 1.0));
        long closeTime = System.currentTimeMillis();
        // 耗时
        long threshold = closeTime - startTime;
        if (log.isInfoEnabled() && threshold > ComplexityUtils.THRESHOLD) {
            log.info("The computation took {} ms", threshold);
        }
        return ComplexityUtils.build(score, structureScore);
    }

    public static ComplexityMode config(WorkflowTask workTask, ComplexityMode complexity) throws Exception {
        workTask.putMetadata(ComplexityUtils.KEY_COMPLEXITY, complexity);
        return complexity;
    }

    public static ComplexityMode result(WorkflowTask workTask) throws Exception {
        ComplexityMode lastMode = workTask.getMetadata(ComplexityUtils.KEY_COMPLEXITY, ComplexityMode.class);
        if (lastMode != null) {
            if (MapUtils.getBoolean(workTask.getMetadata(), ComplexityUtils.KEY_COMPLEXITY_UPGRADE, false)) {
                if (ComplexityMode.TASK_PLANNING.is(lastMode)) {
                    return ComplexityMode.DEEP_THINKING;
                }
                if (ComplexityMode.FAST_REPLY.is(lastMode)) {
                    return ComplexityMode.TASK_PLANNING;
                }
            }
            return lastMode;
        }
        lastMode = ComplexityUtils.score(ComplexityUtils.buildQuery(workTask));
        // 如果开启了Thinking，那么最小级别为TASK_PLANNING
        lastMode = lastMode.getScore() >= ComplexityMode.TASK_PLANNING.getScore() ? lastMode : (ComplexityUtils.isThinking(workTask) ? ComplexityMode.TASK_PLANNING : lastMode);
        workTask.putMetadata(ComplexityUtils.KEY_COMPLEXITY, lastMode);
        return lastMode;
    }

    public static ComplexityMode build(double total, double structure) throws Exception {
        if (total > ComplexityMode.DEEP_THINKING.getScore() || (total > ComplexityMode.TASK_PLANNING.getScore() && structure > 0.7)) {
            return ComplexityMode.DEEP_THINKING;
        }
        if (total > ComplexityMode.FAST_REPLY.getScore()) {
            return ComplexityMode.TASK_PLANNING;
        }
        return ComplexityMode.FAST_REPLY;
    }

    public static String buildQuery(WorkflowTask workTask) throws Exception {
        StringBuffer buffer = new StringBuffer();
        // 获取所有上下文记录用户Query累加来判断复杂度
        if (!CollectionUtils.isEmpty(workTask.getHistories())) {
            for (History current : workTask.getHistories()) {
                // 是用户提问则追加
                if (current.isRole(History.ROLE_USER) && current.isType(History.TYPE_QUERY)) {
                    buffer.append(current.getContent()).append(System.lineSeparator());
                }
            }
        }
        buffer.append(workTask.getOriginal());
        return buffer.toString();
    }

    public static Boolean isThinking(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureField.KEY_THINKING, false);
    }

    public static void resetUpgrade(WorkflowTask workTask) throws Exception {
        workTask.delMetadata(ComplexityUtils.KEY_COMPLEXITY_UPGRADE);
    }

    public static void markUpgrade(WorkflowTask workTask) throws Exception {
        workTask.putMetadata(ComplexityUtils.KEY_COMPLEXITY_UPGRADE, true);
    }
}