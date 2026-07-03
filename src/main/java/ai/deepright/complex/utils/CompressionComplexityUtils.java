package ai.deepright.complex.utils;

import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.GzipUtils;

// Lempel-Ziv (LZ)
public class CompressionComplexityUtils {

    public static double score(String input) throws Exception {
        int original = BytesUtils.utf8Bytes(input);
        if (original < 10) return 0.5;
        // 短文本失去统计意义，给中间值
        byte[] compressed = GzipUtils.compress(input);
        // 注意：应该是 compress
        double ratio = (double) compressed.length / original;
        // 压缩比越接近 1.0（甚至超过 1.0），说明信息越复杂，难以压缩
        return Math.min(ratio, 1.0);
    }
}
