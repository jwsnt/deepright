package ai.deepright.plan;

import java.util.HashMap;
import java.util.Map;

// 将规划中的全角替换为半角，防止更新时无法命中
public class PlanEncode {

    public static final Map<Character, Character> MAP = new HashMap<>(64);

    static {
        // CJK 标点
        PlanEncode.MAP.put('\uFF0C', ',');  PlanEncode.MAP.put('\u3002', '.'); PlanEncode.MAP.put('\uFF1B', ';');
        PlanEncode.MAP.put('\uFF1A', ':');  PlanEncode.MAP.put('\uFF01', '!'); PlanEncode.MAP.put('\uFF1F', '?');
        PlanEncode.MAP.put('\uFF08', '(');  PlanEncode.MAP.put('\uFF09', ')');
        PlanEncode.MAP.put('\u3010', '[');  PlanEncode.MAP.put('\u3011', ']');
        PlanEncode.MAP.put('\u300A', '<');  PlanEncode.MAP.put('\u300B', '>');
        PlanEncode.MAP.put('\u300C', '[');  PlanEncode.MAP.put('\u300D', ']');
        PlanEncode.MAP.put('\u300E', '[');  PlanEncode.MAP.put('\u300F', ']');
        PlanEncode.MAP.put('\u3001', ',');  PlanEncode.MAP.put('\u3000', ' ');
        // 全角 ASCII 符号
        PlanEncode.MAP.put('\uFF5B', '{');  PlanEncode.MAP.put('\uFF5D', '}');
        PlanEncode.MAP.put('\uFF5E', '~');  PlanEncode.MAP.put('\uFF5C', '|');
        PlanEncode.MAP.put('\uFF3C', '\\'); PlanEncode.MAP.put('\uFF0F', '/');
        PlanEncode.MAP.put('\uFF0B', '+');  PlanEncode.MAP.put('\uFF0D', '-');
        PlanEncode.MAP.put('\uFF1D', '=');  PlanEncode.MAP.put('\uFF06', '&');
        PlanEncode.MAP.put('\uFF20', '@');  PlanEncode.MAP.put('\uFF03', '#');
        PlanEncode.MAP.put('\uFF04', '$');  PlanEncode.MAP.put('\uFF05', '%');
        PlanEncode.MAP.put('\uFF3E', '^');  PlanEncode.MAP.put('\uFF0A', '*');
        PlanEncode.MAP.put('\uFF02', '"');  PlanEncode.MAP.put('\uFF07', '\'');
        // 中文引号 / 省略号
        PlanEncode.MAP.put('\u201C', '"');  PlanEncode.MAP.put('\u201D', '"');
        PlanEncode.MAP.put('\u2018', '\''); PlanEncode.MAP.put('\u2019', '\'');
        PlanEncode.MAP.put('\u2026', '.');
    }

    public static String replace(String content) throws Exception {
        char[] chars = content.toCharArray();
        for (int i = 0; i < chars.length; i++) {
            Character half = PlanEncode.MAP.get(chars[i]);
            if (half != null) {
                chars[i] = half;
            }
        }
        return new String(chars);
    }
}
