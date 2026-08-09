package ai.deepright.complex.utils;

import java.util.BitSet;

public class TransitionMatrixUtils {

    protected static final int CHARACTER_SPACE = Character.MAX_VALUE + 1;

    public static double score(String input) throws Exception {
        if (input == null || input.length() < 50) return 0.0;
        // 只记录 ASCII 或常用字符的跳转
        // 使用稀疏统计思想：BitSet[currentChar] 记录可达 nextChar
        BitSet[] transitions = new BitSet[CHARACTER_SPACE];
        int[] touched = new int[Math.min(input.length() - 1, CHARACTER_SPACE)];
        int touchedCount = 0;
        for (int i = 0; i < input.length() - 1; i++) {
            char current = input.charAt(i);
            BitSet nextStates = transitions[current];
            if (nextStates == null) {
                nextStates = new BitSet();
                transitions[current] = nextStates;
                touched[touchedCount++] = current;
            }
            nextStates.set(input.charAt(i + 1));
        }
        // 计算平均每个字符后能接多少种不同的字符
        double totalNextStates = 0.0;
        for (int i = 0; i < touchedCount; i++) {
            totalNextStates += transitions[touched[i]].cardinality();
        }
        double avgNextStates = touchedCount == 0 ? 0.0 : totalNextStates / touchedCount;
        // 如果一个字符后平均接 1.5 到 5 个字符，说明语法结构强
        // 如果太多（接近随机）或太少（死循环），则复杂度低
        if (avgNextStates > 1.2 && avgNextStates < 6.0) return 1.0;
        return 0.3;
    }
}
