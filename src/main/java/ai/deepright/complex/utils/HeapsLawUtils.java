package ai.deepright.complex.utils;

public class HeapsLawUtils {

    public static double score(String input) throws Exception {
        if (input.length() < 10) return 0.0;
        double n = input.length();
        double v = input.chars().distinct().count();
        // Beta = log(V) / log(N)
        double beta = Math.log(v) / Math.log(n);
        // 自然语言的 beta 通常在 0.5 到 0.7 之间
        // 我们衡量它与理想区间 [0.3, 0.8] 的接近程度
        if (beta < 0.3) return 0.2; // 极度重复
        if (beta > 0.8) return 0.4; // 过于杂乱（如随机字符）
        return 1.0; // 落在理想区间，说明具有人类语言的分布特征
    }
}