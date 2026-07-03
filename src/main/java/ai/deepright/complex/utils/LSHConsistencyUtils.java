package ai.deepright.complex.utils;

import java.util.ArrayList;
import java.util.List;

public class LSHConsistencyUtils {

    public static double score(String input) throws Exception {
        if (input.length() < 64) return 0.0;
        int windowSize = 16;
        List<Long> fingerPrints = new ArrayList<>();
        for (int i = 0; i <= input.length() - windowSize; i += windowSize) {
            fingerPrints.add(hash(input.substring(i, i + windowSize)));
        }
        if (fingerPrints.size() < 2) return 0.0;
        double totalDistance = 0;
        for (int i = 0; i < fingerPrints.size() - 1; i++) {
            totalDistance += Long.bitCount(fingerPrints.get(i) ^ fingerPrints.get(i + 1));
        }
        double avgDistance = totalDistance / (fingerPrints.size() - 1);
        // 汉明距离在 10-25 之间通常意味着“有联系但非重复”，即高信息量
        return (avgDistance > 10 && avgDistance < 30) ? 1.0 : 0.4;
    }

    private static long hash(String s) {
        long h = 1125899906842597L; // prime
        for (int i = 0; i < s.length(); i++) {
            h = 31 * h + s.charAt(i);
        }
        return h;
    }
}