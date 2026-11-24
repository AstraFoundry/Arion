# iOS keyboard long-press alternates

Source of `KeyLayout.alternates` in ArionKit. Recorded on the iOS 26.4 simulator (iPhone 17) with the system Simplified Chinese Pinyin keyboard, holding each key and reading the popup left to right. The popup never repeats the held key.

## Where alternates come from

- librime has no long-press data. Rime frontends define their own: Trime per key in `trime.yaml` (`long_click`, `swipe_up`…), Hamster as swipes in `hamster.yaml` (`swipe.keyboardSwipe`). Neither is an iOS-style popup list.
- KeyboardKit 9.9.2 (the last MIT release; 10.0 and later are closed source) has an English list in `BaseCalloutService`. It is a multilingual superset (for example ǎ, ğ, ř, ≠) and does not match what the system keyboard shows, so it is not used.

## English letters (EN mode)

| Key | Alternates |
| --- | --- |
| a | à á â ä æ ã å ā |
| c | ç ć č |
| e | è é ê ë ē ė ę |
| i | î ï í ī į ì |
| l | ł |
| n | ñ ń |
| o | ô ö ò ó œ ø ō õ |
| s | ß ś š |
| u | û ü ù ú ū |
| y | ÿ |
| z | ž ź ż |

q w r t p d g k have none. The Pinyin letters have none.

## Symbols

| Key | Alternates | Pages |
| --- | --- | --- |
| 0 | ° | 123 |
| - | – — • | 123 |
| / | \ | 123 |
| $ | ₽ ¥ € ¢ £ ₩ | English 123, Chinese #+= |
| ¥ | $ € £ | Chinese 123, English #+= |
| & | § | English 123, Chinese #+= |
| " | „ “ ” « » | English 123 |
| . | … | 123 |
| ? ？ | ¿ | 123 |
| ! ！ | ¡ | 123 |
| ' | ` ‘ ’ | English 123, Chinese #+= |
| % | ‰ | #+= |
| 。 | … | Chinese 123 |
| ， | 、 | Chinese 123, Chinese #+= |
| “ | 「 『 | Chinese 123 |
| ” | 」 』 | Chinese 123 |

Keys checked with no popup: : ; ( ) @ , 、 ： ； （ ）, and on #+= 【 】 { } # ^ * + = _ — \ | ～ 《 》 · … ^_^ € £ •.

Not checked: the system US English keyboard (only the Chinese keyboard's EN mode), digits 1–9, and ? ! on the #+= pages.
