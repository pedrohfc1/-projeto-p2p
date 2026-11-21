import {useEffect} from 'react';
import type {SoundVoiceSnapshot} from './callSounds';
import type {ScreenControlState} from './callPresentation';
import {callSounds} from './soundEffects';

export function prepareCallSounds() {
 callSounds.prepare();
}

export function useCallSounds(view: SoundVoiceSnapshot, screen: ScreenControlState, output = '') {
 useEffect(() => { callSounds.output(output); }, [output]);
 useEffect(() => { callSounds.observe(view, screen); }, [view, screen]);
 const joined = !!view?.joined;
 useEffect(() => {
  if (!joined) return;
  const resumeOnInteraction = () => callSounds.prepare();
  window.addEventListener('pointerdown', resumeOnInteraction, true);
  window.addEventListener('keydown', resumeOnInteraction, true);
  return () => {
   window.removeEventListener('pointerdown', resumeOnInteraction, true);
   window.removeEventListener('keydown', resumeOnInteraction, true);
  };
 }, [joined]);
}
