**Aplikacja „UniQuest” (Character Profiler)**

Cel projektu: Stworzenie interaktywnej aplikacji mobilnej, która na podstawie aktywności użytkownika w świecie
rzeczywistym (wydziały uczelni/stoiska) buduje jego profil kompetencyjny w formie karty postaci RPG.

1. Moduł Onboarding i Personalizacja
Rejestracja: Szybkie logowanie (e-mail/social media).
Tworzenie Awatara: Funkcja wykonania selfie, które zostaje przetworzone przez filtr graficzny na styl postaci z gry
(np. pixel art, rzut izometryczny).
Interface: UI stylizowany na panel statystyk bohatera (HP, Mana, Siła, Inteligencja itp. – na początku wyzerowane).

2. Pętla Rozgrywki (User Journey)
Użytkownik odwiedza fizyczne lokalizacje (np. 6 wydziałów, hala sportowa, biblioteka).
Interakcja: Uczestnictwo w prezentacji, warsztacie lub pokazie.
Walidacja aktywności (Action Trigger): Na końcu każdego wydarzenia wyświetlany jest unikalny kod QR.
Zeskanowanie kodu w aplikacji potwierdza obecność.
Moduł Oceny (Feedback Loop): Po skanowaniu użytkownik określa poziom zainteresowania tematem za pomocą
suwaka/skali (np. „Na ile to było dla Ciebie fajne?”).

3. Mechanika Profilowania (Backend Logic)
System przypisuje każdą lokalizację do konkretnych archetypów postaci. Przykładowo:
Wydział Mechaniczny -> Atrybuty: Mechaniczny / Inżynieryjny.
Wydział Informatyki -> Atrybuty: Technologiczny / Logiczny.
Biblioteka/Administracja -> Atrybuty: Korporacyjny / Research.
Hala Sportowa -> Atrybuty: Witalność / Praca Zespołowa.
Algorytm: System sumuje punkty z lokalizacji pomnożone przez subiektywną ocenę użytkownika. Im wyższa ocena
warsztatu mechanicznego, tym silniej postać ewoluuje w stronę „Inżyniera”.

4. Moduł Wynikowy: Karta Postaci
Po zebraniu wymaganej liczby aktywności (np. min. 5 skanów), aplikacja generuje finalny feedback:
Dynamiczny opis: „Twoja ścieżka to w 70% Konstruktor, w 30% Analityk”.
Graficzna Karta Postaci: Atrakcyjna wizualnie karta z podsumowaniem statystyk, którą można pobrać lub
udostępnić w social mediach.
Rekomendacje: Sugestie kierunków studiów lub kół naukowych dopasowanych do uzyskanego profilu.

5. Wymagania Techniczne (MVP):
Moduły: Aparat (skaner QR + selfie), Baza danych (Firebase/Supabase), System powiadomień.
Platformy: iOS / Android (np. Flutter lub React Native).
Design: Estetyka RPG (stylizowany pasek postępu, ikony atrybutów).