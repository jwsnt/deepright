package ai.deepright.complex.utils;

// 字符集的类型多样性
public class CharsetEntropyUtils {

    public static double score(String input) throws Exception {

        boolean hasPunctuation = false;

        boolean hasSymbol = false;

        boolean hasLetter = false;

        boolean hasDigit = false;

        boolean hasMark = false;

        for (int i = 0; i < input.length(); ) {
            int codePoint = input.codePointAt(i);
            int type = Character.getType(codePoint);
            switch (type) {
                case Character.UPPERCASE_LETTER:
                case Character.LOWERCASE_LETTER:
                case Character.TITLECASE_LETTER:
                case Character.MODIFIER_LETTER:
                case Character.OTHER_LETTER:
                    hasLetter = true;
                    break;
                case Character.DECIMAL_DIGIT_NUMBER:
                case Character.LETTER_NUMBER:
                case Character.OTHER_NUMBER:
                    hasDigit = true;
                    break;
                case Character.DASH_PUNCTUATION:
                case Character.START_PUNCTUATION:
                case Character.END_PUNCTUATION:
                case Character.CONNECTOR_PUNCTUATION:
                case Character.OTHER_PUNCTUATION:
                    hasPunctuation = true;
                    break;
                case Character.MATH_SYMBOL:
                case Character.CURRENCY_SYMBOL:
                case Character.MODIFIER_SYMBOL:
                case Character.OTHER_SYMBOL:
                    hasSymbol = true;
                    break;
                case Character.NON_SPACING_MARK:
                case Character.COMBINING_SPACING_MARK:
                case Character.ENCLOSING_MARK:
                    hasMark = true;
                    break;
            }
            if (hasLetter && hasDigit && hasPunctuation && hasSymbol && hasMark) {
                break;
            }
            i += Character.charCount(codePoint);
        }
        int score = 0;
        if (hasLetter) score++;
        if (hasDigit) score++;
        if (hasPunctuation) score++;
        if (hasSymbol) score++;
        if (hasMark) score++;
        // 归一化得分
        return score / 5.0;
    }
}
