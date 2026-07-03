package ai.deepright.complex.utils;

import lombok.extern.slf4j.Slf4j;

@Slf4j
public class BlockDynamicUtils {

    public static int getDynamicBlockSize(int length) {
        return Math.max(64, Math.min(length / 10, 512));
    }

    public static double score(String input) throws Exception {
        int len = input.length();
        if (len < 64) {
            return 0.0;
        }
        int blockSize = BlockDynamicUtils.getDynamicBlockSize(len);
        int blockCount = ((len - blockSize) / blockSize) + 1;
        double[] entropies = new double[blockCount];
        int entropyCount = 0;
        double sum = 0.0;
        for (int i = 0; i <= len - blockSize; i += blockSize) {
            try {
                double entropy = ShannonEntropyUtils.score(input, i, i + blockSize);
                entropies[entropyCount++] = entropy;
                sum += entropy;
            } catch (Exception e) {
                log.warn("Entropy calculation failed for block at {}", i);
            }
        }
        if (entropyCount == 0) return 0.0;
        double avg = sum / entropyCount;
        double varianceSum = 0.0;
        for (int i = 0; i < entropyCount; i++) {
            double delta = entropies[i] - avg;
            varianceSum += delta * delta;
        }
        return Math.sqrt(varianceSum / entropyCount);
    }
}
