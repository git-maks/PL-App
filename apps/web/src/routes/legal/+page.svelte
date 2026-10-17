<script lang="ts">
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
</script>

<svelte:head><title>Zasady gry i prywatność · UniQuest</title></svelte:head>

<TopAppBar title="Zasady i prywatność" back={href('/')} />
<article class="page prose body-medium">
	<p class="muted">Wersja robocza do akceptacji przez Politechnikę Lubelską i Inspektora Ochrony Danych.</p>

	<h2 class="title-large">Zasady gry</h2>
	<ol>
		<li>UniQuest jest bezpłatną grą edukacyjną towarzyszącą wydarzeniom na kampusie Politechniki Lubelskiej.</li>
		<li>Punkty zdobywa się, skanując kod QR wyświetlany na stacji pod koniec zajęć i oceniając zajęcia w skali 1–5. Każdą stację zalicza się raz.</li>
		<li>Kody rotują co 30 sekund. Przesyłanie zdjęć kodów osobom nieobecnym jest niezgodne z zasadami gry.</li>
		<li>Profil postaci i rekomendacje kierunków mają charakter informacyjny i nie wpływają na rekrutację na studia.</li>
		<li>Pseudonimy i nazwy drużyn nie mogą naruszać dóbr osobistych ani zawierać treści obraźliwych – organizator może je zmienić.</li>
	</ol>

	<h2 class="title-large">Prywatność</h2>
	<h3 class="title-medium">Jakie dane przetwarzamy</h3>
	<ul>
		<li><strong>Konto gościa:</strong> losowy identyfikator, pseudonim, zaliczone stacje i oceny, członkostwo w drużynie. Bez imienia, nazwiska i e-maila.</li>
		<li><strong>Awatar:</strong> wyłącznie obrazek pixel-art 48×48 px. Zdjęcie (selfie) przetwarzamy tylko na Twoim urządzeniu i nigdy go nie wysyłamy.</li>
		<li><strong>E-mail (opcjonalnie):</strong> tylko do logowania jednorazowym kodem i przeniesienia postępu na inne urządzenie. Podanie e-maila wymaga ukończenia 16 lat albo zgody opiekuna.</li>
		<li><strong>Powiadomienia push (opcjonalnie):</strong> techniczny adres subskrypcji przeglądarki.</li>
		<li><strong>Cookie sesji:</strong> jedno, niezbędne do działania konta. Nie używamy cookies reklamowych ani zewnętrznych narzędzi śledzących.</li>
	</ul>
	<h3 class="title-medium">Cel i czas przechowywania</h3>
	<p>Dane służą wyłącznie do prowadzenia gry. Uczelnia otrzymuje jedynie zagregowane, anonimowe statystyki (np. średnia ocena stacji). Konta gości nieaktywne przez 12 miesięcy są automatycznie usuwane.</p>
	<h3 class="title-medium">Twoje prawa</h3>
	<p>W ustawieniach możesz w każdej chwili pobrać wszystkie swoje dane i trwale usunąć konto. Administratorem danych jest Politechnika Lubelska, ul. Nadbystrzycka 38D, 20-618 Lublin.</p>

	<h2 class="title-large">Dostępność</h2>
	<p>Aplikacja jest projektowana zgodnie z WCAG 2.1 na poziomie AA. Kod stacji można wpisać ręcznie zamiast skanować, a animacje można wyłączyć w ustawieniach.</p>
</article>

<style>
	.prose {
		line-height: 1.6;
	}
	.prose h2 {
		margin-top: 16px;
	}
	.prose ol,
	.prose ul {
		margin: 0;
		padding-left: 22px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
</style>
