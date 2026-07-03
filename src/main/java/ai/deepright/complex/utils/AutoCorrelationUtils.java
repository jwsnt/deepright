package ai.deepright.complex.utils;

import org.apache.commons.lang3.StringUtils;

public class AutoCorrelationUtils {

    public static final Integer THRESHOLD = Integer.valueOf(StringUtils.defaultIfEmpty(System.getenv("COMPLEXITY_THRESHOLD"), "1000"));

    public static double score(String input) throws Exception {
        if (input.length() > AutoCorrelationUtils.THRESHOLD) return 0.5;
        int maxLag = Math.min(input.length() / 2, 40);
        if (maxLag < 5) return 0.0;
        double maxCorrelation = 0;
        // 计算不同位移下的匹配度
        for (int lag = 1; lag <= maxLag; lag++) {
            int matches = 0;
            int comparisons = input.length() - lag;
            for (int i = 0; i < comparisons; i++) {
                if (input.charAt(i) == input.charAt(i + lag)) {
                    matches++;
                }
            }
            double correlation = (double) matches / comparisons;
            maxCorrelation = Math.max(maxCorrelation, correlation);
        }
        // 如果最大相关性在 0.1 到 0.4 之间，说明既有结构又不死板（典型的逻辑文本）
        // 如果相关性极高 (>0.8)，说明是大量重复
        if (maxCorrelation > 0.1 && maxCorrelation < 0.5) return 1.0;
        return 0.2;
    }
}