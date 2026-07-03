package ai.deepright.complex.utils;

import org.apache.commons.lang3.StringUtils;

public class EnclosureUtils {

    public static double score(String input) throws Exception {
        int pairs = 0;
        String[] targets = {"{", "[", "(", "<", "\"", "`"};
        for (String t : targets) {
            pairs += StringUtils.countMatches(input, t);
        }
        return Math.min(pairs / (input.length() / 20.0), 1.0);
    }
}