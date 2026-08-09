package ai.deepright.complex;

import lombok.AllArgsConstructor;
import lombok.Getter;

@Getter
@AllArgsConstructor
public enum ComplexityMode {

    DEEP_THINKING("DEEP_THINKING", 0.65D),

    TASK_PLANNING("TASK_PLANNING", 0.45D),

    FAST_REPLY("FAST_REPLY", 0.35D);

    private final String code;

    private final Double score;

    public boolean is(ComplexityMode... mode) {
        for (ComplexityMode m : mode) {
            if (this.equals(m)) {
                return true;
            }
        }
        return false;
    }

    // 检查得分是否超过指定阈值
    public boolean exceeds(double score) {
        return this.score >= score;
    }
}